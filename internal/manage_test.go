package internal_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"t0ast.cc/tbml/internal"
	uio "t0ast.cc/tbml/util/io"
)

var uc = "userChrome.css"
var uj = "user.js"

func getConfigurationFixture() internal.Configuration {
	return internal.Configuration{
		Profiles: []internal.ProfileConfiguration{
			{
				ExtensionFiles: []string{
					"extensions/foobar@t0ast.cc.xpi",
				},
				Label:          "test",
				UserChromeFile: &uc,
				UserJSFile:     &uj,
			},
		},
	}
}

func getConfigurationFixtureWithMoreProfiles() internal.Configuration {
	return internal.Configuration{
		Profiles: []internal.ProfileConfiguration{
			{
				ExtensionFiles: []string{
					"extensions/foobar@t0ast.cc.xpi",
				},
				Label:          "test",
				UserChromeFile: &uc,
				UserJSFile:     &uj,
			},
			{
				Label: "test-other",
			},
		},
	}
}

func getProfileInstancesFixture() []internal.ProfileInstance {
	ul2 := "test-usage"
	up2 := 1234
	return []internal.ProfileInstance{
		{
			Created:       time.Date(2021, 10, 24, 18, 12, 1, 289350236, time.UTC),
			InstanceLabel: "test-1",
			LastUsed:      time.Date(2021, 10, 24, 18, 12, 13, 382409155, time.UTC),
			ProfileLabel:  "test",
		},
		{
			Created:       time.Date(2021, 10, 25, 18, 12, 1, 289350236, time.UTC),
			InstanceLabel: "test-2",
			LastUsed:      time.Date(2021, 10, 25, 18, 12, 13, 382409155, time.UTC),
			ProfileLabel:  "test",
			UsageLabel:    &ul2,
			UsagePID:      &up2,
		},
	}
}

func setUpProfilesWithAbsolutePath(t *testing.T) (internal.Configuration, func()) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "tbml-test-*")
	assert.NoError(t, err)
	assert.NoError(t, uio.CopyDir("testdata/instances/profiles", tmpDir))

	config := getConfigurationFixture()
	config.ProfilePath = tmpDir

	return config, func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}
}

func TestReadConfiguration(t *testing.T) {
	testCases := []struct {
		desc string

		configFileName  string
		prepareExpected func(expected *internal.Configuration)
	}{
		{
			desc: "No profile path",

			configFileName: "config-no-profile-path.json",
			prepareExpected: func(expected *internal.Configuration) {
				cache, err := os.UserCacheDir()
				assert.NoError(t, err)
				expected.ProfilePath = filepath.Join(cache, "tbml")
			},
		},
		{
			desc: "Profile path from home",

			configFileName: "config-profile-path-from-home.json",
			prepareExpected: func(expected *internal.Configuration) {
				home, err := os.UserHomeDir()
				assert.NoError(t, err)
				expected.ProfilePath = filepath.Join(home, ".tbml")
			},
		},
		{
			desc: "Profile path from root",

			configFileName: "config-profile-path-from-root.json",
			prepareExpected: func(expected *internal.Configuration) {
				expected.ProfilePath = "/tmp/tbml"
			},
		},
		{
			desc: "Relative profile path",

			configFileName: "config-relative-profile-path.json",
			prepareExpected: func(expected *internal.Configuration) {
				expected.ProfilePath = "testdata/tbml/profiles"
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			expected := getConfigurationFixture()
			tC.prepareExpected(&expected)

			config, configDir, err := internal.ReadConfiguration(filepath.Join("testdata", tC.configFileName))
			assert.NoError(t, err)
			assert.Equal(t, expected, config)
			assert.Equal(t, "testdata", configDir)
		})
	}
}

func TestReadConfigurationNonexistent(t *testing.T) {
	_, _, err := internal.ReadConfiguration("testdata/config-nonexistent.json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestGetProfileInstances(t *testing.T) {
	config := getConfigurationFixture()
	config.ProfilePath = "testdata/instances/profiles"

	actual, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)

	expected := getProfileInstancesFixture()
	assert.Equal(t, expected, actual)
}

func TestGetProfileInstancesAbsolute(t *testing.T) {
	config, cleanup := setUpProfilesWithAbsolutePath(t)
	defer cleanup()

	actual, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)

	expected := getProfileInstancesFixture()
	assert.Equal(t, expected, actual)
}

func TestGetProfileInstance(t *testing.T) {
	config := getConfigurationFixture()
	config.ProfilePath = "testdata/instances/profiles"

	actual, err := internal.GetProfileInstance(config, "test-2")
	assert.NoError(t, err)

	expected := getProfileInstancesFixture()[1]
	assert.Equal(t, "test-2", expected.InstanceLabel)
	assert.Equal(t, expected, actual)
}

func TestGetProfileInstanceAbsolute(t *testing.T) {
	config, cleanup := setUpProfilesWithAbsolutePath(t)
	defer cleanup()

	actual, err := internal.GetProfileInstance(config, "test-2")
	assert.NoError(t, err)

	expected := getProfileInstancesFixture()[1]
	assert.Equal(t, "test-2", expected.InstanceLabel)
	assert.Equal(t, expected, actual)
}

func TestDeleteInstance(t *testing.T) {
	config, cleanup := setUpProfilesWithAbsolutePath(t)
	defer cleanup()

	instancesBefore, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)
	assert.Len(t, instancesBefore, 2)

	assert.NoError(t, internal.DeleteInstance(config, instancesBefore[0]))

	instancesAfter, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)
	assert.Equal(t, instancesBefore[1:], instancesAfter)
}

func TestDeleteInstanceInUse(t *testing.T) {
	config, cleanup := setUpProfilesWithAbsolutePath(t)
	defer cleanup()

	instancesBefore, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)
	assert.Len(t, instancesBefore, 2)

	err = internal.DeleteInstance(config, instancesBefore[1])
	assert.ErrorIs(t, err, internal.ErrInstanceInUse)

	instancesAfter, err := internal.GetProfileInstances(config)
	assert.NoError(t, err)
	assert.Equal(t, instancesBefore, instancesAfter)
}

func TestFindProfileByLabel(t *testing.T) {
	config := getConfigurationFixtureWithMoreProfiles()
	assert.Len(t, config.Profiles, 2)

	actual := internal.FindProfileByLabel(config, "test")

	assert.Equal(t, &config.Profiles[0], actual)
}

func TestFindProfileByLabelNonexistent(t *testing.T) {
	config := getConfigurationFixtureWithMoreProfiles()
	assert.Len(t, config.Profiles, 2)

	actual := internal.FindProfileByLabel(config, "nonexistent")

	assert.Nil(t, actual)
}

func TestGetProfileLabels(t *testing.T) {
	config := getConfigurationFixtureWithMoreProfiles()

	actual := internal.GetProfileLabels(config)

	assert.Equal(t, []string{"test", "test-other"}, actual)
}

func TestGetTopics(t *testing.T) {
	instances := getProfileInstancesFixture()

	actual := internal.GetTopics(instances)

	assert.Equal(t, []string{"test-usage"}, actual)
}

func TestFindInstanceByTopic(t *testing.T) {
	instances := getProfileInstancesFixture()

	assert.Nil(t, internal.FindInstanceByTopic(instances, "unused-topic-label"))
	assert.Equal(t, instances[1], *internal.FindInstanceByTopic(instances, "test-usage"))
}

func TestGetBestInstance(t *testing.T) {
	testCases := []struct {
		desc string

		expectedBestInstance internal.ProfileInstance
		instances            []internal.ProfileInstance
	}{
		{
			desc: "Choose only free instance",

			expectedBestInstance: getProfileInstancesFixture()[0],
			instances:            getProfileInstancesFixture(),
		},
		{
			desc: "Choose oldest instance",

			expectedBestInstance: internal.ProfileInstance{
				InstanceLabel: "oldest-instance",
				Created:       time.UnixMilli(0),
				ProfileLabel:  "test",
			},
			instances: append(getProfileInstancesFixture(), internal.ProfileInstance{
				InstanceLabel: "oldest-instance",
				Created:       time.UnixMilli(0),
				ProfileLabel:  "test",
			}),
		},
		{
			desc: "Create new instance",

			expectedBestInstance: internal.ProfileInstance{
				InstanceLabel: "test-1",
				ProfileLabel:  "test",
			},
			instances: []internal.ProfileInstance{},
		},
		{
			desc: "Create new instance with incremented number in label",

			expectedBestInstance: internal.ProfileInstance{
				InstanceLabel: "test-3",
				ProfileLabel:  "test",
			},
			instances: getProfileInstancesFixture()[1:],
		},
		{
			desc: "Skip instances of other profiles",

			expectedBestInstance: getProfileInstancesFixture()[0],
			instances: append(getProfileInstancesFixture(), internal.ProfileInstance{
				InstanceLabel: "oldest-instance",
				Created:       time.UnixMilli(0),
				ProfileLabel:  "test-other",
			}),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			config := getConfigurationFixture()
			assert.Equal(t, config.Profiles[0].Label, "test")

			actual := internal.GetBestInstance(config.Profiles[0], tC.instances)

			assert.Equal(t, tC.expectedBestInstance, actual)
		})
	}
}
