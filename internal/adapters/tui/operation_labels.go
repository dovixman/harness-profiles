package tui

func operationTitle(op operation) string {
	switch op {
	case opAdd:
		return "Add Harness"
	case opUpdate:
		return "Update Harness"
	case opDeleteHarness:
		return "Delete Harness"
	case opSwitch:
		return "Switch Profile"
	case opAdopt:
		return "Adopt Profile"
	case opCreateProfile:
		return "Create Profile"
	case opRenameProfile:
		return "Update Profile"
	case opClone:
		return "Clone Profile"
	case opDeleteProfile:
		return "Delete Profile"
	case opAddLink:
		return "Add Managed Link"
	case opUpdateLink:
		return "Update Managed Link"
	case opDeleteLink:
		return "Delete Managed Link"
	}
	return "Operation"
}

func operationProgress(op operation) string {
	switch op {
	case opAdd:
		return "Adding harness..."
	case opUpdate:
		return "Updating harness..."
	case opDeleteHarness:
		return "Deleting harness..."
	case opSwitch:
		return "Switching profile..."
	case opAdopt:
		return "Adopting profile..."
	case opCreateProfile:
		return "Creating profile..."
	case opRenameProfile:
		return "Updating profile..."
	case opClone:
		return "Cloning profile..."
	case opDeleteProfile:
		return "Deleting profile..."
	case opAddLink:
		return "Adding managed link..."
	case opUpdateLink:
		return "Updating managed link..."
	case opDeleteLink:
		return "Removing managed link..."
	}
	return "Working..."
}
