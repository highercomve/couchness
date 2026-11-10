package common

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/highercomve/couchness/utils"
)

// Movie version grouping.
//
// Jellyfin (and Emby/Plex) show several files as versions of one movie only
// when they share a folder named "Title (Year)" and each file is named
// "Title (Year) - <label>.<ext>". Downloads land wherever the torrent put
// them, so copies of the same movie end up as separate entries. This planner
// finds those copies and moves them into that layout, carrying subtitles,
// .nfo files, extras folders and artwork along. It never deletes or
// overwrites files.

// DefaultMinMovieSize filters out samples and promo clips.
const DefaultMinMovieSize int64 = 100 << 20

// GroupOptions configures PlanMovieGroups.
type GroupOptions struct {
	// MinSize is the smallest file treated as a movie (default 100 MiB).
	MinSize int64
	// Busy reports whether a path belongs to a torrent that is still in the
	// download client; it returns the torrent name. Nil disables the check.
	Busy func(path string) (string, bool)
	// Force plans groups even when one of their files is busy.
	Force bool
}

// FileMove is one planned rename.
type FileMove struct {
	From string
	To   string
}

// MovieVersion is one copy of a movie and everything that moves with it.
type MovieVersion struct {
	Path       string
	Size       int64
	Label      string
	Dest       string
	Companions []FileMove // subtitles, nfo, extras folders, artwork
	Leftovers  []string   // files in the old folder that stay where they are
	sourceDir  string     // dedicated folder the copy came from, "" for loose files
}

// MovieGroup is a set of files that are the same movie.
type MovieGroup struct {
	Title    string
	Year     int
	Folder   string
	Versions []*MovieVersion
	Skip     string // reason the group is not moved; empty when it will be
}

// GroupPlan is the result of PlanMovieGroups.
type GroupPlan struct {
	Root   string
	Groups []*MovieGroup
}

// Moves returns every rename of groups that are not skipped, in order.
func (p *GroupPlan) Moves() []FileMove {
	var out []FileMove
	for _, g := range p.Groups {
		if g.Skip != "" {
			continue
		}
		for _, v := range g.Versions {
			if v.Path != v.Dest {
				out = append(out, FileMove{From: v.Path, To: v.Dest})
			}
			for _, c := range v.Companions {
				if c.From != c.To {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

var (
	movieExtensions = map[string]bool{
		".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true, ".wmv": true,
		".mpg": true, ".mpeg": true, ".ts": true, ".m2ts": true, ".webm": true,
	}
	subtitleExts = map[string]bool{
		".srt": true, ".ass": true, ".ssa": true, ".sub": true, ".idx": true, ".vtt": true, ".sup": true, ".smi": true,
	}
	imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	artNames  = map[string]bool{
		"poster": true, "folder": true, "cover": true, "fanart": true, "backdrop": true,
		"banner": true, "logo": true, "clearlogo": true, "landscape": true, "thumb": true, "disc": true,
	}
	// Folders Jellyfin treats as extras of the movie they sit next to.
	extrasDirs = map[string]bool{
		"featurettes": true, "extras": true, "behind the scenes": true, "deleted scenes": true,
		"interviews": true, "scenes": true, "shorts": true, "trailers": true, "other": true, "clips": true,
	}
	skipDirs = map[string]bool{"sample": true, "samples": true, "video_ts": true, "bdmv": true, "subs": true, "subtitles": true}

	partRe       = regexp.MustCompile(`(?i)(?:^|[ ._\-\[(])(?:cd|disc|disk|dvd|part|pt)[ ._-]?\d{1,2}(?:$|[ ._\-\])])`)
	resolutionRe = regexp.MustCompile(`(?i)\b(2160p|1440p|1080p|720p|576p|480p)\b`)
	fourKRe      = regexp.MustCompile(`(?i)\b(4k|uhd)\b`)
	sourceRe     = regexp.MustCompile(`(?i)\b(blu-?ray|bdrip|brrip|bdremux|remux|web-?dl|web-?rip|web|hdrip|dvdrip|dvd|hdtv|hdcam|camrip|cam|telesync)\b`)
	codecRe      = regexp.MustCompile(`(?i)\b(x265|x264|h\.?265|h\.?264|hevc|avc|xvid|divx|av1|vp9)\b`)
	hdrRe        = regexp.MustCompile(`(?i)\b(hdr10\+|hdr10|hdr|dv|dovi|dolby[ .]?vision)\b`)
	bitDepthRe   = regexp.MustCompile(`(?i)\b10[ .-]?bits?\b`)
	audioRe      = regexp.MustCompile(`(?i)\b(dts-hd(?:[ .]ma)?|dts-x|dts|truehd|atmos|ddp|dd\+|eac3|ac3|aac|dd|flac|opus|mp3)(?:[ .]?([1-7]\.[01]))?\b`)
	groupRe      = regexp.MustCompile(`-\s?([A-Za-z0-9]{2,})$`)
	bracketTagRe = regexp.MustCompile(`\[([^\]]+)\]$`)
	unsafeChars  = regexp.MustCompile(`[/\\:*?"<>|]+`)
	spacesRe     = regexp.MustCompile(`\s+`)
	leadingNumRe = regexp.MustCompile(`^\d+_`)
	yearOnlyRe   = regexp.MustCompile(`^\d{4}$`)
)

type movieFile struct {
	path  string
	size  int64
	title string
	year  int
	key   string
	part  bool
}

// PlanMovieGroups finds movies stored more than once under root and plans
// the moves that make Jellyfin show them as versions of a single movie.
func PlanMovieGroups(root string, opts GroupOptions) (*GroupPlan, error) {
	if opts.MinSize <= 0 {
		opts.MinSize = DefaultMinMovieSize
	}
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	files, err := scanMovieFiles(root, opts.MinSize)
	if err != nil {
		return nil, err
	}
	videosPerDir := map[string]int{}
	for _, f := range files {
		for d := filepath.Dir(f.path); d != root && strings.HasPrefix(d, root); d = filepath.Dir(d) {
			videosPerDir[d]++
		}
	}

	byKey := map[string][]*movieFile{}
	var keys []string
	for _, f := range files {
		if _, ok := byKey[f.key]; !ok {
			keys = append(keys, f.key)
		}
		byKey[f.key] = append(byKey[f.key], f)
	}
	sort.Strings(keys)

	plan := &GroupPlan{Root: root}
	planned := map[string]bool{} // destinations already claimed
	for _, key := range keys {
		members := byKey[key]
		if len(members) < 2 {
			continue
		}
		g := buildGroup(root, members, videosPerDir, planned, opts)
		plan.Groups = append(plan.Groups, g)
	}
	return plan, nil
}

func scanMovieFiles(root string, minSize int64) ([]*movieFile, error) {
	var out []*movieFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		name := d.Name()
		if d.IsDir() {
			lower := strings.ToLower(name)
			if path != root && (strings.HasPrefix(name, ".") || skipDirs[lower] || extrasDirs[lower]) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !movieExtensions[ext] || isSampleName(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() < minSize {
			return nil
		}
		title, year := parseMovieName(path, root)
		if title == "" {
			return nil
		}
		out = append(out, &movieFile{
			path: path, size: info.Size(), title: title, year: year,
			key:  normalizeTitle(title) + "|" + fmt.Sprint(year),
			part: partRe.MatchString(strings.TrimSuffix(name, filepath.Ext(name))),
		})
		return nil
	})
	return out, err
}

func isSampleName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "sample") || strings.Contains(lower, ".sample.") ||
		strings.Contains(lower, "-sample.") || strings.HasPrefix(lower, "rarbg")
}

// parseMovieName reads title and year from the file name, falling back to
// the enclosing folder (e.g. "Title (2020) [1080p]/file.mkv").
func parseMovieName(path, root string) (string, int) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	candidates := []string{base}
	if parent := filepath.Dir(path); parent != root {
		candidates = append(candidates, filepath.Base(parent))
	}
	bestTitle, bestYear := "", 0
	for _, c := range candidates {
		t, err := utils.ParseTorrent(c)
		if err != nil {
			continue
		}
		title := cleanTitle(t.Title)
		if title == "" {
			continue
		}
		if t.Year > 0 {
			return title, t.Year
		}
		if bestTitle == "" {
			bestTitle = title
		}
	}
	return bestTitle, bestYear
}

func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if !strings.Contains(s, " ") {
		s = strings.ReplaceAll(s, ".", " ")
	}
	s = strings.Trim(s, " .-[](){}")
	return spacesRe.ReplaceAllString(s, " ")
}

// normalizeTitle makes "The Dead Don't Die" and "The.Dead.Dont.Die" equal.
func normalizeTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func buildGroup(root string, members []*movieFile, videosPerDir map[string]int, planned map[string]bool, opts GroupOptions) *MovieGroup {
	sort.Slice(members, func(i, j int) bool { return members[i].size > members[j].size })
	g := &MovieGroup{Title: bestTitle(members), Year: members[0].year}
	g.Folder = filepath.Join(root, folderName(g.Title, g.Year))

	prefix := folderName(g.Title, g.Year)
	for _, m := range members {
		v := &MovieVersion{Path: m.path, Size: m.size, Label: versionLabel(m.path)}
		parent := filepath.Dir(m.path)
		base := strings.TrimSuffix(filepath.Base(m.path), filepath.Ext(m.path))
		switch {
		case parent == g.Folder:
			// already grouped: keep its name so repeated runs change nothing
			if strings.HasPrefix(base, prefix+" - ") {
				v.Label = strings.TrimPrefix(base, prefix+" - ")
			}
		case parent != root && videosPerDir[parent] == 1:
			v.sourceDir = parent
		}
		g.Versions = append(g.Versions, v)
	}
	for _, m := range members {
		if m.part {
			g.Skip = "multi-part movie (cd1/cd2...); Jellyfin needs those named as parts, not versions"
			return g
		}
	}
	if info, err := os.Stat(g.Folder); err == nil && !info.IsDir() {
		g.Skip = fmt.Sprintf("%s exists and is not a folder", g.Folder)
		return g
	}
	if opts.Busy != nil && !opts.Force {
		for _, v := range g.Versions {
			if name, busy := opts.Busy(v.Path); busy {
				g.Skip = fmt.Sprintf("still in the download client (torrent %q); moving it would break seeding. Remove the torrent or use --force", name)
				return g
			}
		}
	}

	uniqueLabels(g.Versions)
	for _, v := range g.Versions {
		ext := strings.ToLower(filepath.Ext(v.Path))
		v.Dest = claim(filepath.Join(g.Folder, prefix+" - "+v.Label+ext), v.Path, planned)
		planCompanions(g, v, planned)
	}
	return g
}

func bestTitle(members []*movieFile) string {
	best := members[0].title
	for _, m := range members {
		// prefer the spelling with punctuation ("Don't") and real spaces
		if len(m.title) > len(best) || (strings.ContainsAny(m.title, "'’") && !strings.ContainsAny(best, "'’")) {
			best = m.title
		}
	}
	return best
}

func folderName(title string, year int) string {
	name := strings.TrimSpace(spacesRe.ReplaceAllString(unsafeChars.ReplaceAllString(title, " "), " "))
	if year > 0 {
		name = fmt.Sprintf("%s (%d)", name, year)
	}
	return name
}

// versionLabel builds a short, readable label from the release name, e.g.
// "1080p BluRay x264 DTS-HD PublicHD".
func versionLabel(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	names := []string{base}
	if parent := filepath.Base(filepath.Dir(path)); parent != "" {
		names = append(names, parent)
	}
	find := func(re *regexp.Regexp) string {
		for _, n := range names {
			if m := re.FindStringSubmatch(strings.ReplaceAll(n, "_", " ")); m != nil {
				return m[0]
			}
		}
		return ""
	}
	var parts []string
	res := strings.ToLower(find(resolutionRe))
	if res == "" && find(fourKRe) != "" {
		res = "2160p"
	}
	add := func(s string) {
		if s != "" {
			parts = append(parts, s)
		}
	}
	add(res)
	add(normalizeSource(find(sourceRe)))
	add(normalizeCodec(find(codecRe)))
	if hdr := hdrLabel(names); hdr != "" {
		add(hdr)
	} else if find(bitDepthRe) != "" {
		add("10bit")
	}
	add(normalizeAudio(find(audioRe)))
	add(releaseGroup(base))
	label := strings.Join(dedupe(parts), " ")
	label = strings.TrimSpace(spacesRe.ReplaceAllString(unsafeChars.ReplaceAllString(label, " "), " "))
	if len(label) > 60 {
		label = strings.TrimSpace(label[:60])
	}
	if label == "" {
		label = "Version"
	}
	return label
}

// hdrLabel reports Dolby Vision and the HDR10 flavour together ("DV HDR10").
func hdrLabel(names []string) string {
	dv, hdr := false, ""
	for _, n := range names {
		for _, m := range hdrRe.FindAllString(strings.ReplaceAll(n, "_", " "), -1) {
			switch u := strings.ToUpper(strings.NewReplacer(".", "", " ", "").Replace(m)); u {
			case "DV", "DOVI", "DOLBYVISION":
				dv = true
			default:
				if len(u) > len(hdr) {
					hdr = u // HDR10+ > HDR10 > HDR
				}
			}
		}
		if dv || hdr != "" {
			break
		}
	}
	switch {
	case dv && hdr != "":
		return "DV " + hdr
	case dv:
		return "DV"
	}
	return hdr
}

func normalizeSource(s string) string {
	switch strings.ToLower(strings.ReplaceAll(s, "-", "")) {
	case "":
		return ""
	case "bluray":
		return "BluRay"
	case "bdrip", "brrip":
		return "BDRip"
	case "bdremux", "remux":
		return "Remux"
	case "webdl", "web":
		return "WEB-DL"
	case "webrip":
		return "WEBRip"
	case "hdrip":
		return "HDRip"
	case "dvdrip", "dvd":
		return "DVDRip"
	case "hdtv":
		return "HDTV"
	default:
		return strings.ToUpper(s)
	}
}

func normalizeCodec(s string) string {
	switch strings.ToLower(strings.ReplaceAll(s, ".", "")) {
	case "":
		return ""
	case "x265", "h265", "hevc":
		return "HEVC"
	case "x264", "h264", "avc":
		return "H264"
	case "xvid", "divx":
		return "XviD"
	default:
		return strings.ToUpper(s)
	}
}

func normalizeAudio(s string) string {
	if s == "" {
		return ""
	}
	u := strings.ToUpper(strings.ReplaceAll(s, " ", "."))
	u = strings.Replace(u, "DTS-HD.MA.", "DTS-HD MA ", 1)
	u = strings.Replace(u, "DTS-HD.MA", "DTS-HD MA", 1)
	return u
}

// releaseGroup extracts "PublicHD" from "...x264-PublicHD", "ExYu - SRB"
// from "...[ExYu - SRB]" and "anoXmous" from "...x264.anoXmous_".
func releaseGroup(base string) string {
	b := strings.TrimRight(base, "_. ")
	if m := bracketTagRe.FindStringSubmatch(b); m != nil {
		tag := strings.TrimSpace(m[1])
		if !strings.Contains(strings.ToLower(tag), "yts") && !resolutionRe.MatchString(tag) {
			return strings.ReplaceAll(tag, " - ", "-")
		}
		b = strings.TrimSpace(b[:len(b)-len(m[0])])
	}
	if m := groupRe.FindStringSubmatch(b); m != nil && !isKnownToken(m[1]) {
		return m[1]
	}
	tokens := strings.FieldsFunc(b, func(r rune) bool { return r == '.' || r == ' ' })
	if n := len(tokens); n > 0 {
		last := tokens[n-1]
		if isAlnum(last) && len(last) >= 3 && !isKnownToken(last) {
			return last
		}
	}
	return ""
}

func isKnownToken(s string) bool {
	for _, re := range []*regexp.Regexp{resolutionRe, fourKRe, sourceRe, codecRe, hdrRe, bitDepthRe, audioRe} {
		if re.MatchString(s) {
			return true
		}
	}
	return yearOnlyRe.MatchString(s)
}

func isAlnum(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return s != ""
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		k := strings.ToLower(s)
		if !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	return out
}

// uniqueLabels makes labels distinct inside a group by appending the size.
func uniqueLabels(versions []*MovieVersion) {
	count := map[string]int{}
	for _, v := range versions {
		count[strings.ToLower(v.Label)]++
	}
	for _, v := range versions {
		if count[strings.ToLower(v.Label)] > 1 {
			v.Label = fmt.Sprintf("%s %.1fGB", v.Label, float64(v.Size)/(1<<30))
		}
	}
	seen := map[string]int{}
	for _, v := range versions {
		k := strings.ToLower(v.Label)
		seen[k]++
		if seen[k] > 1 {
			v.Label = fmt.Sprintf("%s (%d)", v.Label, seen[k])
		}
	}
}

// claim reserves dest, adding " (2)", " (3)"... when it is taken on disk or
// by an earlier planned move. A file already at dest keeps it.
func claim(dest, current string, planned map[string]bool) string {
	ext := filepath.Ext(dest)
	stem := strings.TrimSuffix(dest, ext)
	candidate := dest
	for i := 2; ; i++ {
		_, err := os.Lstat(candidate)
		free := errors.Is(err, os.ErrNotExist) || candidate == current
		if free && !planned[candidate] {
			planned[candidate] = true
			return candidate
		}
		candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
	}
}

// planCompanions moves subtitles, .nfo files, extras folders and artwork
// that belong to v next to its new location.
func planCompanions(g *MovieGroup, v *MovieVersion, planned map[string]bool) {
	newBase := strings.TrimSuffix(v.Dest, filepath.Ext(v.Dest))
	dir := filepath.Dir(v.Path)
	base := strings.TrimSuffix(filepath.Base(v.Path), filepath.Ext(v.Path))
	taken := map[string]bool{v.Path: true}

	// 1. Files named after the video: "<base>.en.srt", "<base>_eng.srt", "<base>.nfo".
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			full := filepath.Join(dir, name)
			ext := strings.ToLower(filepath.Ext(name))
			if full == v.Path || !(subtitleExts[ext] || ext == ".nfo") {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
				continue
			}
			suffix := normalizeSuffix(name[len(base):])
			v.Companions = append(v.Companions, FileMove{From: full, To: claim(newBase+suffix, full, planned)})
			taken[full] = true
		}
	}
	if v.sourceDir == "" {
		return // loose file next to other movies: only name-matched files are ours
	}

	// 2. The copy has its own folder: other subtitles inside it are its too.
	_ = filepath.WalkDir(v.sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if path != v.sourceDir && (lower == "sample" || lower == "samples" || extrasDirs[lower]) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if taken[path] || !subtitleExts[ext] {
			return nil
		}
		stem := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		stem = leadingNumRe.ReplaceAllString(stem, "")
		v.Companions = append(v.Companions, FileMove{From: path, To: claim(newBase+"."+stem+ext, path, planned)})
		taken[path] = true
		return nil
	})

	// 3. Extras folders and artwork sitting directly in that folder.
	if entries, err := os.ReadDir(v.sourceDir); err == nil {
		for _, e := range entries {
			full := filepath.Join(v.sourceDir, e.Name())
			lower := strings.ToLower(e.Name())
			if e.IsDir() && extrasDirs[lower] {
				dest := filepath.Join(g.Folder, e.Name())
				if !planned[dest] && !exists(dest) {
					planned[dest] = true
					v.Companions = append(v.Companions, FileMove{From: full, To: dest})
					taken[full] = true
				}
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if !e.IsDir() && imageExts[ext] && artNames[strings.TrimSuffix(lower, ext)] {
				dest := filepath.Join(g.Folder, strings.TrimSuffix(lower, ext)+ext)
				if !planned[dest] && !exists(dest) {
					planned[dest] = true
					v.Companions = append(v.Companions, FileMove{From: full, To: dest})
					taken[full] = true
				}
			}
		}
	}

	// 4. Everything else stays and is reported.
	_ = filepath.WalkDir(v.sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		for t := range taken {
			if path == t || strings.HasPrefix(path, t+string(os.PathSeparator)) {
				return nil
			}
		}
		v.Leftovers = append(v.Leftovers, path)
		return nil
	})
}

// normalizeSuffix turns "_eng.srt" / "-en.srt" / " en.srt" into ".eng.srt".
func normalizeSuffix(s string) string {
	s = strings.TrimLeft(s, "_- ")
	if !strings.HasPrefix(s, ".") {
		s = "." + s
	}
	return s
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// GroupResult summarises ApplyMovieGroups.
type GroupResult struct {
	Moved      int
	Failed     []error
	RemovedDir []string
}

// ApplyMovieGroups performs the plan. Each rename is checked again right
// before it happens; existing destinations are never overwritten. Folders a
// version came from are removed only when they end up completely empty.
func ApplyMovieGroups(plan *GroupPlan, logf func(format string, args ...any)) GroupResult {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	var res GroupResult
	for _, g := range plan.Groups {
		if g.Skip != "" {
			continue
		}
		if err := os.MkdirAll(g.Folder, 0o775); err != nil {
			res.Failed = append(res.Failed, fmt.Errorf("%s: %w", g.Folder, err))
			continue
		}
		for _, v := range g.Versions {
			moves := v.Companions
			if v.Path != v.Dest {
				moves = append([]FileMove{{From: v.Path, To: v.Dest}}, v.Companions...)
			}
			videoFailed := false
			for i, m := range moves {
				if videoFailed {
					break // keep companions with their video if the video stayed
				}
				if err := safeRename(m.From, m.To); err != nil {
					res.Failed = append(res.Failed, err)
					logf("  ! %v", err)
					if i == 0 && v.Path != v.Dest {
						videoFailed = true
					}
					continue
				}
				res.Moved++
				logf("  moved %s\n     -> %s", rel(plan.Root, m.From), rel(plan.Root, m.To))
			}
			if v.sourceDir != "" && !videoFailed {
				res.RemovedDir = append(res.RemovedDir, removeEmptyDirs(v.sourceDir, plan.Root)...)
			}
		}
	}
	return res
}

func safeRename(from, to string) error {
	if from == to {
		return nil
	}
	if _, err := os.Lstat(from); err != nil {
		return fmt.Errorf("%s: %w", from, err)
	}
	if exists(to) {
		return fmt.Errorf("%s already exists, not overwriting", to)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o775); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("moving %s: %w", from, err)
	}
	return nil
}

// removeEmptyDirs deletes dir and its subfolders only if they contain no
// files at all, never going above root.
func removeEmptyDirs(dir, root string) []string {
	var removed []string
	var walk func(string) bool
	walk = func(d string) bool {
		entries, err := os.ReadDir(d)
		if err != nil {
			return false
		}
		empty := true
		for _, e := range entries {
			if e.IsDir() && walk(filepath.Join(d, e.Name())) {
				continue
			}
			empty = false
		}
		if empty && d != root && strings.HasPrefix(d, root+string(os.PathSeparator)) {
			if os.Remove(d) == nil {
				removed = append(removed, d)
				return true
			}
		}
		return false
	}
	walk(dir)
	return removed
}

func rel(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}
