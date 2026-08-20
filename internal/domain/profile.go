package domain

type Profile struct {
	HarnessID string
	Name      string
	Path      string
}

func ValidateProfileName(name string) error {
	return validateName("profile name", name)
}
