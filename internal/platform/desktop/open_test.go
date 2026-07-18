package desktop

import "testing"

func TestManagementURLAndValidation(t *testing.T) {
	managementURL, err := ManagementURL("3210")
	if err != nil || managementURL != "http://127.0.0.1:3210" {
		t.Fatalf("ManagementURL = %q, %v", managementURL, err)
	}
	if err := validateManagementURL(managementURL); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
	for _, value := range []string{
		"https://127.0.0.1:3210", "http://localhost:3210", "http://127.0.0.1",
		"http://127.0.0.1:3210/path", "http://127.0.0.1:3210?next=evil",
		"http://user@127.0.0.1:3210", "http://127.0.0.1:70000",
	} {
		if err := validateManagementURL(value); err == nil {
			t.Errorf("unsafe URL accepted: %q", value)
		}
	}
}

func TestManagementURLRejectsInvalidPort(t *testing.T) {
	for _, value := range []string{"", "0", "65536", "abc"} {
		if _, err := ManagementURL(value); err == nil {
			t.Errorf("invalid port accepted: %q", value)
		}
	}
}
