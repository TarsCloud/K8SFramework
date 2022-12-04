package tool

func HasFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func RemoveFinalizer(finalizers []string, finalizer string) []string {
	var newFinalizers []string
	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}
	return newFinalizers
}
