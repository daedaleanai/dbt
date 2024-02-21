package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const flagsFileName = "FLAGS.json"

var (
	flagValues      = getFlagValues()
	flagsLocked     = false
	registeredFlags = map[string]flagInterface{}
)

type flagInfo struct {
	Description   string
	Type          string
	AllowedValues []string
	Value         string
}

type flagInterface interface {
	info() flagInfo
	setFromString(string)
	setToDefault() bool
}

type StringFlag struct {
	Name          string
	Description   string
	AllowedValues []string
	DefaultFn     func() string

	isInitialized bool
	value         string
}

func (flag *StringFlag) Value() string {
	initializeFlag(flag, flag.Name, &flag.isInitialized)
	return flag.value
}

func (flag StringFlag) Register() *StringFlag {
	initializeFlag(&flag, flag.Name, &flag.isInitialized)
	return &flag
}

func (flag *StringFlag) info() flagInfo {
	return flagInfo{flag.Description, "string", flag.AllowedValues, flag.value}
}

func (flag *StringFlag) setFromString(value string) {
	flag.value = value
}

func (flag *StringFlag) setToDefault() bool {
	if flag.DefaultFn != nil {
		flag.value = flag.DefaultFn()
		return true
	}
	return false
}

type BoolFlag struct {
	Name        string
	Description string
	DefaultFn   func() bool

	isInitialized bool
	value         bool
}

func (flag *BoolFlag) Value() bool {
	initializeFlag(flag, flag.Name, &flag.isInitialized)
	return flag.value
}

func (flag BoolFlag) Register() *BoolFlag {
	initializeFlag(&flag, flag.Name, &flag.isInitialized)
	return &flag
}

func (flag *BoolFlag) info() flagInfo {
	return flagInfo{flag.Description, "bool", []string{"true", "false"}, strconv.FormatBool(flag.value)}
}

func (flag *BoolFlag) setFromString(value string) {
	switch value {
	case "true":
		flag.value = true
	case "false":
		flag.value = false
	default:
		Fatal("invalid value '%s' for boolean flag '%s'", value, flag.Name)
	}
}

func (flag *BoolFlag) setToDefault() bool {
	if flag.DefaultFn != nil {
		flag.value = flag.DefaultFn()
		return true
	}
	return false
}

type IntFlag struct {
	Name          string
	Description   string
	AllowedValues []int64
	DefaultFn     func() int64

	isInitialized bool
	value         int64
}

func (flag *IntFlag) Value() int64 {
	initializeFlag(flag, flag.Name, &flag.isInitialized)
	return flag.value
}

func (flag IntFlag) Register() *IntFlag {
	initializeFlag(&flag, flag.Name, &flag.isInitialized)
	return &flag
}

func (flag *IntFlag) info() flagInfo {
	allowedValues := []string{}
	for _, value := range flag.AllowedValues {
		allowedValues = append(allowedValues, strconv.FormatInt(value, 10))
	}
	return flagInfo{flag.Description, "int", allowedValues, strconv.FormatInt(flag.value, 10)}
}

func (flag *IntFlag) setFromString(value string) {
	i64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		Fatal("invalid value '%s' for integer flag '%s': %s", value, flag.Name, err)
	}
	flag.value = i64
}

func (flag *IntFlag) setToDefault() bool {
	if flag.DefaultFn != nil {
		flag.value = flag.DefaultFn()
		return true
	}
	return false
}

type FloatFlag struct {
	Name        string
	Description string
	DefaultFn   func() float64

	isInitialized bool
	value         float64
}

func (flag *FloatFlag) Value() float64 {
	initializeFlag(flag, flag.Name, &flag.isInitialized)
	return flag.value
}

func (flag FloatFlag) Register() *FloatFlag {
	initializeFlag(&flag, flag.Name, &flag.isInitialized)
	return &flag
}

func (flag *FloatFlag) info() flagInfo {
	return flagInfo{flag.Description, "float", []string{}, strconv.FormatFloat(flag.value, 'f', -1, 64)}
}

func (flag *FloatFlag) setFromString(value string) {
	f64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		Fatal("invalid value '%s' for floating-point flag '%s': %s", value, flag.Name, err)
	}
	flag.value = f64
}

func (flag *FloatFlag) setToDefault() bool {
	if flag.DefaultFn != nil {
		flag.value = flag.DefaultFn()
		return true
	}
	return false
}

func initializeFlag(flag flagInterface, name string, isInitialized *bool) {
	if *isInitialized {
		return
	}

	if flagsLocked {
		Fatal("flag '%s' accessed, but not reistered", name)
	}

	*isInitialized = true
	if _, exists := registeredFlags[name]; exists {
		Fatal("multiple flags with name '%s'", name)
	}
	registeredFlags[name] = flag

	if value, exists := flagValues[name]; exists {
		flag.setFromString(value)
	} else if !flag.setToDefault() {
		Fatal("flag '%s' has no value", name)
	}

	info := flag.info()
	if len(info.AllowedValues) == 0 {
		return
	}
	for _, value := range info.AllowedValues {
		if info.Value == value {
			return
		}
	}
	Fatal("flag '%s' has disallowed value '%s'", name, info.Value)
}

func lockAndGetFlags(storePersistedFlags bool) map[string]flagInfo {
	flagsLocked = true

	flagInfo := map[string]flagInfo{}
	flagValues := map[string]string{}
	for name, flag := range registeredFlags {
		info := flag.info()
		flagInfo[name] = info
		flagValues[name] = info.Value
	}

	if storePersistedFlags {
		// Store current flag values in FLAGS.json file.
		data, err := json.Marshal(flagValues)
		if err != nil {
			Fatal("failed to marshal config flag values: %s", err)
		}
		flagsFilePath := path.Join(input.OutputDir, flagsFileName)
		if err := os.MkdirAll(filepath.Dir(flagsFilePath), os.ModePerm); err != nil {
			Fatal("Failed to create directory for flags file: %s", err)
		}
		err = ioutil.WriteFile(flagsFilePath, data, fileMode)
		if err != nil {
			Fatal("failed to write config flag values: %s", err)
		}
	}

	return flagInfo
}

func getFlagValues() map[string]string {
	mergedFlags := map[string]string{}

	// Copy over flags from the workspace MODULE file.
	for name, value := range input.WorkspaceFlags {
		mergedFlags[name] = value
	}

	// Read flags from the FLAGS.json file. Flags from that file will override
	// flags from the workspace MODULE file.
	persistedFlags := map[string]string{}
	data, err := ioutil.ReadFile(path.Join(input.OutputDir, flagsFileName))
	if err == nil {
		err = json.Unmarshal(data, &persistedFlags)
		if err != nil {
			Fatal("failed to unmarshall config flags: %s", err)
		}

		for name, value := range persistedFlags {
			mergedFlags[name] = value
		}
	} else if !os.IsNotExist(err) {
		Fatal("failed to read config flags: %s", err)
	}

	// Command-line flags have the highest priority and will override all other flags.
	for name, value := range input.CmdlineFlags {
		mergedFlags[name] = value
	}

	return mergedFlags
}
