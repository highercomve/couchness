package telegram

import (
	"strings"
	"testing"

	"github.com/highercomve/couchness/models"
)

func TestHelpTextRespectsRole(t *testing.T) {
	viewerHelp := helpText(models.TelegramRoleViewer)
	if strings.Contains(viewerHelp, "/download") || strings.Contains(viewerHelp, "/invite") {
		t.Fatalf("viewer received privileged commands: %s", viewerHelp)
	}

	userHelp := helpText(models.TelegramRoleUser)
	if !strings.Contains(userHelp, "/download") || strings.Contains(userHelp, "/invite") {
		t.Fatalf("user commands are incorrect: %s", userHelp)
	}

	ownerHelp := helpText(models.TelegramRoleOwner)
	if !strings.Contains(ownerHelp, "/download") || !strings.Contains(ownerHelp, "/invite") || !strings.Contains(ownerHelp, "/add_show") || !strings.Contains(ownerHelp, "/add_movie") || !strings.Contains(ownerHelp, "/remove_show") {
		t.Fatalf("owner commands are incomplete: %s", ownerHelp)
	}
}

func TestRandomCode(t *testing.T) {
	first, err := randomCode()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomCode()
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("invalid random invite codes: %q %q", first, second)
	}
}
