package common

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// mkfile creates a sparse file of the given size (no real disk usage).
func mkfile(t *testing.T, root, rel string, size int64) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if size > 0 {
		if err := f.Truncate(size); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	return p
}

const gb = int64(1) << 30

// libraryLikeRockPi mirrors real folder layouts seen in /mnt/cloud/movies.
func libraryLikeRockPi(t *testing.T) string {
	root := t.TempDir()
	// Inside Llewyn Davis: three copies in three layouts.
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD/Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD.mkv", 10*gb)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD/Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD.nfo", 1088)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.anoXmous/Inside.Llewyn.Davis.2013.1080p.BluRay.x264.anoXmous_.mp4", 2*gb)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.anoXmous/Inside.Llewyn.Davis.2013.1080p.BluRay.x264.anoXmous_eng.srt", 98553)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.[ExYu - SRB].mp4", gb+gb/2)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.[ExYu - SRB].en.srt", 98226)
	mkfile(t, root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.[ExYu - SRB].es.srt", 91011)
	// The Dead Don't Die: different spellings, both loose.
	mkfile(t, root, "The Dead Don't Die (2019) (1080p BluRay x265 HEVC 10bit DDP5.1) - WiCK.mkv", 3*gb)
	mkfile(t, root, "The Dead Dont Die.2019.DVDRip.XviD.AC3-EVO...---...avi", gb)
	mkfile(t, root, "The Dead Dont Die.2019.DVDRip.XviD.AC3-EVO...---...en.srt", 1000)
	// Strays: YTS folder with Subs/ and junk, plus a FLUX folder.
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/Strays.2023.REPACK.1080p.WEBRip.x264.AAC5.1-[YTS.MX].mp4", 2*gb)
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/Strays.2023.REPACK.1080p.WEBRip.x264.AAC5.1-[YTS.MX].srt", 100)
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/Subs/English.srt", 100)
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/Subs/Latin American.spa.srt", 100)
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/www.YTS.MX.jpg", 100)
	mkfile(t, root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]/YTSProxies.com.txt", 100)
	mkfile(t, root, "Strays.2023.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-FLUX[TGx]/Strays.2023.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-FLUX.mkv", 6*gb)
	mkfile(t, root, "Strays.2023.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-FLUX[TGx]/[TGx]Downloaded from torrentgalaxy.to .txt", 100)
	// Two-part movie: must not be treated as versions.
	mkfile(t, root, "Luna de Avellaneda (Moon of Avellaneda) [2004] DvDrip-paTon/Luna de Avellaneda (Moon of Avellaneda) [2004] Cd1-DvDrip-paTon.avi", 700<<20)
	mkfile(t, root, "Luna de Avellaneda (Moon of Avellaneda) [2004] DvDrip-paTon/Luna de Avellaneda (Moon of Avellaneda) [2004] Cd2-DvDrip-paTon.avi", 700<<20)
	// Single movie with a sample, a promo clip and featurettes: untouched.
	mkfile(t, root, "Deadpool and Wolverine 2024 2160p Bluray x265 DDP+Atmos-KiNGDOM/Deadpool and Wolverine 2024 2160p Bluray x265 DDP+Atmos-KiNGDOM.mkv", 13*gb)
	mkfile(t, root, "Deadpool and Wolverine 2024 2160p Bluray x265 DDP+Atmos-KiNGDOM/sample.mkv", 200<<20)
	mkfile(t, root, "Adaptation.2002.1080p.BluRay.H264.AAC-RARBG/Adaptation.2002.1080p.BluRay.H264.AAC-RARBG.mp4", 2*gb)
	mkfile(t, root, "Adaptation.2002.1080p.BluRay.H264.AAC-RARBG/RARBG.com.mp4", 1<<20)
	mkfile(t, root, "The Whale (2022) (1080p BluRay x265 HEVC 10bit AAC 5.1 Tigole)/Featurettes/People Are Amazing - Making The Whale.mkv", 149<<20)
	mkfile(t, root, "The Whale (2022) (1080p BluRay x265 HEVC 10bit AAC 5.1 Tigole)/The Whale (2022) (1080p BluRay x265 10bit Tigole).mkv", 5*gb)
	return root
}

func groupByTitle(t *testing.T, plan *GroupPlan) map[string]*MovieGroup {
	t.Helper()
	out := map[string]*MovieGroup{}
	for _, g := range plan.Groups {
		out[normalizeTitle(g.Title)] = g
	}
	return out
}

func TestPlanMovieGroupsFindsRealDuplicates(t *testing.T) {
	root := libraryLikeRockPi(t)
	plan, err := PlanMovieGroups(root, GroupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	groups := groupByTitle(t, plan)
	if len(groups) != 4 {
		var names []string
		for _, g := range plan.Groups {
			names = append(names, g.Title)
		}
		t.Fatalf("want 4 groups (Llewyn, Dead Don't Die, Strays, Luna), got %d: %v", len(groups), names)
	}

	llewyn := groups["insidellewyndavis"]
	if llewyn == nil || llewyn.Skip != "" {
		t.Fatalf("Inside Llewyn Davis should be planned: %+v", llewyn)
	}
	if llewyn.Year != 2013 || filepath.Base(llewyn.Folder) != "Inside Llewyn Davis (2013)" {
		t.Errorf("folder = %q, year = %d", llewyn.Folder, llewyn.Year)
	}
	if len(llewyn.Versions) != 3 {
		t.Fatalf("want 3 versions, got %d", len(llewyn.Versions))
	}
	labels := map[string]bool{}
	for _, v := range llewyn.Versions {
		if labels[v.Label] {
			t.Errorf("duplicate label %q", v.Label)
		}
		labels[v.Label] = true
		if !strings.HasPrefix(filepath.Base(v.Dest), "Inside Llewyn Davis (2013) - ") {
			t.Errorf("dest %q does not follow Jellyfin's version naming", v.Dest)
		}
		if filepath.Dir(v.Dest) != llewyn.Folder {
			t.Errorf("dest %q not inside %q", v.Dest, llewyn.Folder)
		}
	}
	// largest copy first, with a descriptive label
	if got := llewyn.Versions[0].Label; got != "1080p BluRay H264 DTS-HD MA 5.1 PublicHD" {
		t.Errorf("PublicHD label = %q", got)
	}

	// companions travel with their video and keep a parsable language suffix
	wantSuffixes := map[string]string{
		"anoXmous_eng.srt":    ".eng.srt",
		"[ExYu - SRB].en.srt": ".en.srt",
		"[ExYu - SRB].es.srt": ".es.srt",
		"PublicHD.nfo":        ".nfo",
	}
	found := 0
	for _, v := range llewyn.Versions {
		newBase := strings.TrimSuffix(v.Dest, filepath.Ext(v.Dest))
		for _, c := range v.Companions {
			for src, suffix := range wantSuffixes {
				if strings.HasSuffix(c.From, src) {
					found++
					if c.To != newBase+suffix {
						t.Errorf("%s -> %s, want %s", filepath.Base(c.From), c.To, newBase+suffix)
					}
				}
			}
		}
	}
	if found != len(wantSuffixes) {
		t.Errorf("found %d of %d expected companions", found, len(wantSuffixes))
	}

	dead := groups["thedeaddontdie"]
	if dead == nil || dead.Title != "The Dead Don't Die" || len(dead.Versions) != 2 {
		t.Fatalf("Dead Don't Die not grouped with its apostrophe kept: %+v", dead)
	}

	luna := groups["lunadeavellaneda"]
	if luna == nil || !strings.Contains(luna.Skip, "multi-part") {
		t.Fatalf("Cd1/Cd2 must be skipped as a multi-part movie, got %+v", luna)
	}

	strays := groups["strays"]
	if strays == nil || strays.Skip != "" {
		t.Fatalf("Strays should be planned: %+v", strays)
	}
	var yts *MovieVersion
	for _, v := range strays.Versions {
		if strings.Contains(v.Path, "YTS") {
			yts = v
		}
	}
	if yts == nil {
		t.Fatal("YTS copy missing")
	}
	var subs []string
	for _, c := range yts.Companions {
		subs = append(subs, strings.TrimPrefix(c.To, strings.TrimSuffix(yts.Dest, filepath.Ext(yts.Dest))))
	}
	sort.Strings(subs)
	want := []string{".English.srt", ".Latin American.spa.srt", ".srt"}
	if strings.Join(subs, "|") != strings.Join(want, "|") {
		t.Errorf("YTS subtitles = %v, want %v", subs, want)
	}
	if len(yts.Leftovers) != 2 {
		t.Errorf("junk files should be reported as leftovers, got %v", yts.Leftovers)
	}

	for _, g := range plan.Groups {
		for _, v := range g.Versions {
			if strings.Contains(strings.ToLower(v.Path), "sample") || strings.Contains(v.Path, "RARBG.com") {
				t.Errorf("sample/promo picked up as a version: %s", v.Path)
			}
		}
	}
}

func TestPlanMovieGroupsSkipsSeedingTorrents(t *testing.T) {
	root := libraryLikeRockPi(t)
	busy := func(p string) (string, bool) {
		if strings.Contains(p, "PublicHD") {
			return "Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD", true
		}
		return "", false
	}
	plan, err := PlanMovieGroups(root, GroupOptions{Busy: busy})
	if err != nil {
		t.Fatal(err)
	}
	if g := groupByTitle(t, plan)["insidellewyndavis"]; !strings.Contains(g.Skip, "download client") {
		t.Fatalf("seeding group must be skipped, got %q", g.Skip)
	}
	plan, err = PlanMovieGroups(root, GroupOptions{Busy: busy, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if g := groupByTitle(t, plan)["insidellewyndavis"]; g.Skip != "" {
		t.Fatalf("--force should plan it, got %q", g.Skip)
	}
}

func TestApplyMovieGroupsMovesWithoutDeleting(t *testing.T) {
	root := libraryLikeRockPi(t)
	plan, err := PlanMovieGroups(root, GroupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	moves := plan.Moves()
	res := ApplyMovieGroups(plan, t.Logf)
	if len(res.Failed) != 0 {
		t.Fatalf("failures: %v", res.Failed)
	}
	if res.Moved != len(moves) {
		t.Errorf("moved %d of %d planned", res.Moved, len(moves))
	}
	for _, m := range moves {
		if exists(m.From) {
			t.Errorf("source still there: %s", m.From)
		}
		if !exists(m.To) {
			t.Errorf("destination missing: %s", m.To)
		}
	}

	folder := filepath.Join(root, "Inside Llewyn Davis (2013)")
	entries, _ := os.ReadDir(folder)
	var videos int
	for _, e := range entries {
		if movieExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			videos++
		}
	}
	if videos != 3 {
		t.Errorf("want 3 versions in %s, got %d", folder, videos)
	}

	// folders that became empty are gone; folders with junk stay
	if exists(filepath.Join(root, "Inside.Llewyn.Davis.2013.1080p.BluRay.x264.anoXmous")) {
		t.Error("empty anoXmous folder should be removed")
	}
	if !exists(filepath.Join(root, "Strays (2023) [REPACK] [1080p] [WEBRip] [5.1] [YTS.MX]", "YTSProxies.com.txt")) {
		t.Error("junk must never be deleted")
	}
	// untouched: single movies and the multi-part movie
	for _, rel := range []string{
		"Deadpool and Wolverine 2024 2160p Bluray x265 DDP+Atmos-KiNGDOM/sample.mkv",
		"Luna de Avellaneda (Moon of Avellaneda) [2004] DvDrip-paTon/Luna de Avellaneda (Moon of Avellaneda) [2004] Cd1-DvDrip-paTon.avi",
		"The Whale (2022) (1080p BluRay x265 HEVC 10bit AAC 5.1 Tigole)/Featurettes/People Are Amazing - Making The Whale.mkv",
	} {
		if !exists(filepath.Join(root, rel)) {
			t.Errorf("should be untouched: %s", rel)
		}
	}

	// running again finds nothing left to move
	again, err := PlanMovieGroups(root, GroupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(again.Moves()); n != 0 {
		t.Errorf("second run should be a no-op, plans %d moves: %v", n, again.Moves())
	}
}

func TestApplyNeverOverwrites(t *testing.T) {
	root := libraryLikeRockPi(t)
	plan, err := PlanMovieGroups(root, GroupOptions{})
	if err != nil {
		t.Fatal(err)
	}
	g := groupByTitle(t, plan)["thedeaddontdie"]
	// something appears at the destination between planning and applying
	mkfile(t, root, filepath.Join(filepath.Base(g.Folder), filepath.Base(g.Versions[0].Dest)), 10)
	res := ApplyMovieGroups(plan, nil)
	if len(res.Failed) == 0 {
		t.Fatal("expected a refusal to overwrite")
	}
	if !exists(g.Versions[0].Path) {
		t.Error("source must stay when the destination is taken")
	}
}

func TestVersionLabel(t *testing.T) {
	cases := map[string]string{
		"/m/Avatar.The.Way.of.Water.2022.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR10.HEVC-CMRG/Avatar.The.Way.of.Water.2022.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR10.HEVC-CMRG.mkv": "2160p WEB-DL HEVC DV HDR10 DDP5.1 CMRG",
		"/m/Avatar.The.Way.of.Water.2022.1080p.WEB-DL.DD2.0.H.264-RiGHTNOW/Avatar.The.Way.of.Water.2022.1080p.WEB-DL.DD2.0.H.264-RiGHTNOW.mkv":                       "1080p WEB-DL H264 DD2.0 RiGHTNOW",
		"/m/Nosferatu (2024) [2160p] [4K] [WEB] [5.1] [YTS.MX]/Nosferatu.2024.2160p.4K.WEB.x265.10bit.AAC5.1-[YTS.MX].mkv":                                           "2160p WEB-DL HEVC 10bit AAC5.1",
	}
	for path, want := range cases {
		if got := versionLabel(path); got != want {
			t.Errorf("versionLabel(%s)\n got %q\nwant %q", filepath.Base(path), got, want)
		}
	}
}
