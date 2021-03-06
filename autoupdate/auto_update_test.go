package autoupdate

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gemnasium/toolbelt/config"
	"github.com/gemnasium/toolbelt/models"
)

func TestFetchUpdateSet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		jsonOutput :=
			`{"id":1,"requirement_updates":{"Rubygem": [{"file":{"path":"Gemfile","sha":"dc6bdc865c85a4f5c6ef0f4ba8909d8652fd8cd0"},"patch":"--- Gemfile\n+++ Gemfile\n@@ -5 +5 @@\n-gem \"warden\", \"0.10.3\"\n+gem \"warden\", '~> 1.2.3'\n@@ -4 +4 @@\n-gem \"rails\", \"3.0.0.beta3\"\n+gem \"rails\", '~> 4.0.3'\n@@ -7 +7 @@\n-gem \"webrat\", \"0.7\"\n+gem \"webrat\", '~> 0.7.3'\n"}]},"version_updates":{}}`
		fmt.Fprintln(w, jsonOutput)
	}))
	defer ts.Close()
	config.APIEndpoint = ts.URL
	expectedUpdateSet := &UpdateSet{
		ID: 1,
		RequirementUpdates: map[string][]RequirementUpdate{
			"Rubygem": []RequirementUpdate{
				RequirementUpdate{
					File: models.DependencyFile{
						Path: "Gemfile",
						SHA:  "dc6bdc865c85a4f5c6ef0f4ba8909d8652fd8cd0",
					},
					Patch: "--- Gemfile\n+++ Gemfile\n@@ -5 +5 @@\n-gem \"warden\", \"0.10.3\"\n+gem \"warden\", '~> 1.2.3'\n@@ -4 +4 @@\n-gem \"rails\", \"3.0.0.beta3\"\n+gem \"rails\", '~> 4.0.3'\n@@ -7 +7 @@\n-gem \"webrat\", \"0.7\"\n+gem \"webrat\", '~> 0.7.3'\n",
				},
			},
		},
		VersionUpdates: map[string][]VersionUpdate{},
	}

	resultSet, err := fetchUpdateSet("blah")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(resultSet, expectedUpdateSet) {
		t.Errorf("Expected resultSet to be:\n%#v\nGot:\n%#v\n", expectedUpdateSet, resultSet)
	}
}

func TestRestoreDepFiles(t *testing.T) {
	gemfile := models.DependencyFile{Path: "Gemfile", Content: []byte("Gemfile content")}
	dfiles := []models.DependencyFile{gemfile}
	err := restoreDepFiles(dfiles)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gemfile.Path)
	body, err := ioutil.ReadFile(gemfile.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Gemfile content" {
		t.Errorf("Restored file should have content: 'Gemfile content', got: '%s'", body)

	}
}

func TestRestoreDepFilesWithInvalidPath(t *testing.T) {
	gemfile := models.DependencyFile{Path: "", Content: []byte("Gemfile content")}
	dfiles := []models.DependencyFile{gemfile}
	err := restoreDepFiles(dfiles)
	if err == nil {
		t.Error("restoreDepFiles should fail")
	}
	fmt.Println() // Hack for goconvey
}

func fakeInstaller(reqUpdates []RequirementUpdate, orgDepFiles, uptDepFiles *[]models.DependencyFile) error {
	for _, ru := range reqUpdates {
		var f models.DependencyFile = ru.File
		f.Content = []byte("original content")
		*orgDepFiles = append(*orgDepFiles, f)
		fmt.Println("Patching", f.Path)
		f.Content = []byte("New content")
		*uptDepFiles = append(*uptDepFiles, f)
	}
	return nil
}

func fakeUpdater(versionUpdates []VersionUpdate, orgDepFiles, uptDepFiles *[]models.DependencyFile) error {
	f := models.DependencyFile{Path: "Gemfile.lock", SHA: "09c2f8647e14e49e922b955c194102070597c2d1", Content: []byte("original content")}
	*orgDepFiles = append(*orgDepFiles, f)
	f.Content = []byte("updated content")
	f.SHA = "141162477fd3bf27aed3bbea4fe3d17c71d6c7be"
	*uptDepFiles = append(*uptDepFiles, f)
	return nil
}

func TestApplyUpdateSet(t *testing.T) {
	// register new installer:
	installers["fakePackage"] = fakeInstaller
	updaters["fakePackage"] = fakeUpdater

	updateSet := &UpdateSet{
		ID: 1,
		RequirementUpdates: map[string][]RequirementUpdate{
			"fakePackage": []RequirementUpdate{
				RequirementUpdate{
					File: models.DependencyFile{
						Path: "Gemfile",
						SHA:  "dc6bdc865c85a4f5c6ef0f4ba8909d8652fd8cd0",
					},
					Patch: "--- Gemfile\n+++ Gemfile\n@@ -5 +5 @@\n-gem \"warden\", \"0.10.3\"\n+gem \"warden\", '~> 1.2.3'\n@@ -4 +4 @@\n-gem \"rails\", \"3.0.0.beta3\"\n+gem \"rails\", '~> 4.0.3'\n@@ -7 +7 @@\n-gem \"webrat\", \"0.7\"\n+gem \"webrat\", '~> 0.7.3'\n",
				},
			},
		},
		VersionUpdates: map[string][]VersionUpdate{
			"fakePackage": []VersionUpdate{
				VersionUpdate{
					Package:       models.Package{Name: "aGem", Slug: "aGem", Type: "fakePackage"},
					OldVersion:    "1.2.3",
					TargetVersion: "1.2.5",
				},
			},
		},
	}
	orgDepFiles, uptDepFiles, err := applyUpdateSet(updateSet)
	if err != nil {
		t.Fatal(err)
	}
	expOrgDepFiles := []models.DependencyFile{
		models.DependencyFile{Path: "Gemfile", SHA: "dc6bdc865c85a4f5c6ef0f4ba8909d8652fd8cd0", Content: []byte("original content")},
		models.DependencyFile{Path: "Gemfile.lock", SHA: "09c2f8647e14e49e922b955c194102070597c2d1", Content: []byte("original content")},
	}
	expUptDepFiles := []models.DependencyFile{
		models.DependencyFile{Path: "Gemfile", SHA: "dc6bdc865c85a4f5c6ef0f4ba8909d8652fd8cd0", Content: []byte("New content")},
		models.DependencyFile{Path: "Gemfile.lock", SHA: "141162477fd3bf27aed3bbea4fe3d17c71d6c7be", Content: []byte("updated content")},
	}
	if !reflect.DeepEqual(orgDepFiles, expOrgDepFiles) {
		t.Errorf("Expectd orgDepFiles to be: %#v, got: %#v", expOrgDepFiles, orgDepFiles)
	}
	if !reflect.DeepEqual(uptDepFiles, expUptDepFiles) {
		t.Errorf("Expectd uptDepFiles to be: %#v, got: %#v", expUptDepFiles, uptDepFiles)
	}
}
