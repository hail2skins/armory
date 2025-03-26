package unit

import (
	"io/ioutil"
	"regexp"
	"testing"
)

func TestFormFieldConsistency(t *testing.T) {
	// Test that field names are consistent between forms and controller processing

	// Check for acquired date field name consistency
	newTemplate, err := ioutil.ReadFile("../../cmd/web/views/owner/gun/new.templ")
	if err != nil {
		t.Fatal("Could not read new.templ file:", err)
	}

	editTemplate, err := ioutil.ReadFile("../../cmd/web/views/owner/gun/edit.templ")
	if err != nil {
		t.Fatal("Could not read edit.templ file:", err)
	}

	controller, err := ioutil.ReadFile("../../internal/controller/owner.go")
	if err != nil {
		t.Fatal("Could not read owner.go file:", err)
	}

	// Check that the name attributes in form fields match what the controller expects
	newTemplateStr := string(newTemplate)
	editTemplateStr := string(editTemplate)
	controllerStr := string(controller)

	// In new.templ, the field should have name="acquired"
	newAcquiredFieldRegex := regexp.MustCompile(`name="acquired"`)
	newAcquiredMatch := newAcquiredFieldRegex.MatchString(newTemplateStr)

	// In edit.templ, the field should have name="acquired_date"
	editAcquiredFieldRegex := regexp.MustCompile(`name="acquired_date"`)
	editAcquiredMatch := editAcquiredFieldRegex.MatchString(editTemplateStr)

	// In the controller, what field name is expected?
	controllerRegex := regexp.MustCompile(`acquiredDateStr := c\.PostForm\("([^"]+)"\)`)
	controllerMatches := controllerRegex.FindAllStringSubmatch(controllerStr, -1)

	if len(controllerMatches) < 1 {
		t.Fatal("Could not find date field processing in controller")
	}

	expectedFieldName := controllerMatches[0][1]
	t.Logf("Controller expects field name: %s", expectedFieldName)
	t.Logf("new.templ has field named 'acquired': %v", newAcquiredMatch)
	t.Logf("edit.templ has field named 'acquired_date': %v", editAcquiredMatch)

	// Verify the controller is consistent internally
	for i, match := range controllerMatches {
		if match[1] != expectedFieldName {
			t.Errorf("Controller inconsistency at match %d: found '%s', expected '%s'",
				i, match[1], expectedFieldName)
		}
	}

	// Check for field name mismatches
	if expectedFieldName == "acquired_date" && newAcquiredMatch {
		t.Error("Field name mismatch: new.templ uses 'acquired' but controller expects 'acquired_date'")
	}

	if expectedFieldName == "acquired" && editAcquiredMatch {
		t.Error("Field name mismatch: edit.templ uses 'acquired_date' but controller expects 'acquired'")
	}

	if expectedFieldName != "acquired" && expectedFieldName != "acquired_date" {
		t.Errorf("Unexpected field name in controller: %s", expectedFieldName)
	}
}
