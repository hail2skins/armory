package tests

import (
	"os"
	"regexp"
	"testing"
)

// TestControllerFormFieldConsistency checks that templates use field names that match what controllers expect
func TestControllerFormFieldConsistency(t *testing.T) {
	// Test the assign role form
	controllerPath := "../internal/controller/admin_permissions_controller.go"
	templatePath := "../cmd/web/views/admin/permissions/assign_role.templ"

	// Read files
	controller, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatalf("Could not read controller file: %v", err)
	}

	template, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("Could not read template file: %v", err)
	}

	// Find what field name the controller expects for user ID
	// Look for userID := ctx.PostForm("...")
	controllerStr := string(controller)
	controllerRegex := regexp.MustCompile(`(?m)userID\s*:=\s*ctx\.PostForm\(\s*"([^"]+)"\s*\)`)
	controllerMatches := controllerRegex.FindStringSubmatch(controllerStr)

	if len(controllerMatches) < 2 {
		t.Fatalf("Could not find user ID field in controller")
	}

	expectedFieldName := controllerMatches[1]
	t.Logf("StoreAssignRole controller expects field name: %s", expectedFieldName)

	// Find what field name the template uses
	templateStr := string(template)
	templateRegex := regexp.MustCompile(`<select[^>]*id\s*=\s*["']user_id["'][^>]*name\s*=\s*["']([^"']+)["'][^>]*>`)
	templateMatches := templateRegex.FindStringSubmatch(templateStr)

	if len(templateMatches) < 2 {
		// Try alternative regex
		templateRegex = regexp.MustCompile(`<select[^>]*name\s*=\s*["']([^"']+)["'][^>]*id\s*=\s*["']user_id["'][^>]*>`)
		templateMatches = templateRegex.FindStringSubmatch(templateStr)

		if len(templateMatches) < 2 {
			t.Fatalf("Could not find user select field in template")
		}
	}

	templateFieldName := templateMatches[1]
	t.Logf("Template uses field name: %s", templateFieldName)

	// They should match
	if expectedFieldName != templateFieldName {
		t.Errorf("Field name mismatch! Controller expects '%s' but template uses '%s'",
			expectedFieldName, templateFieldName)
	}

	// Also test the remove user role functionality
	controllerRemoveRegex := regexp.MustCompile(`(?m)user\s*:=\s*ctx\.PostForm\(\s*"([^"]+)"\s*\)`)
	controllerRemoveMatches := controllerRemoveRegex.FindStringSubmatch(controllerStr)

	if len(controllerRemoveMatches) < 2 {
		t.Fatalf("Could not find user field in RemoveUserRole controller")
	}

	expectedRemoveFieldName := controllerRemoveMatches[1]
	t.Logf("RemoveUserRole controller expects field name: %s", expectedRemoveFieldName)

	// Find the remove form in index.templ
	indexPath := "../cmd/web/views/admin/permissions/index.templ"
	indexTemplate, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Could not read index template file: %v", err)
	}

	indexStr := string(indexTemplate)
	removeFormRegex := regexp.MustCompile(`<input[^>]*name\s*=\s*["']([^"']+)["'][^>]*value\s*=\s*["']` + "`" + `\+userRole\.User\+` + "`" + `["'][^>]*>`)
	removeFormMatches := removeFormRegex.FindStringSubmatch(indexStr)

	if len(removeFormMatches) < 2 {
		t.Logf("Warning: Could not find remove user role form in index template")
		t.Logf("You should manually verify that the remove form uses field name '%s'", expectedRemoveFieldName)
	} else {
		templateRemoveFieldName := removeFormMatches[1]
		t.Logf("Remove form uses field name: %s", templateRemoveFieldName)

		if expectedRemoveFieldName != templateRemoveFieldName {
			t.Errorf("Field name mismatch in remove form! Controller expects '%s' but template uses '%s'",
				expectedRemoveFieldName, templateRemoveFieldName)
		}
	}
}
