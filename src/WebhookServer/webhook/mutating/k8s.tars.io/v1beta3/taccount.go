package v1beta3

import (
	"crypto/sha1"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/json"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"regexp"
)

const PasswordPattern = `^[\x21-\x7e]{6,32}$`
const BcryptHashCost = 6

func generateBcryptPassword(in string) ([]byte, error) {
	sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(in)))
	return bcrypt.GenerateFromPassword([]byte(sha1String), BcryptHashCost)
}

const UnsafeTAccountAnnotationKey = "kubectl.kubernetes.io/last-applied-configuration"
const UnsafeTAccountAnnotationPath = "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration"

func mutatingCreateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &tarsAppsV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	var jsonPatch tarsMeta.JsonPatch

	if newTAccount.Spec.Authentication.Password != nil {
		passwordString := *newTAccount.Spec.Authentication.Password
		ok, _ := regexp.MatchString(PasswordPattern, passwordString)
		if !ok {
			err := fmt.Errorf("password should match pattern %s", PasswordPattern)
			return nil, err
		}
		bcryptPassword, _ := generateBcryptPassword(passwordString)
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:   tarsMeta.JsonPatchRemove,
			Path: "/spec/authentication/password",
		})

		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/spec/authentication/bcryptPassword",
			Value: string(bcryptPassword),
		})
	}

	tokens := make([]tarsAppsV1beta3.TAccountAuthenticationToken, 0, 0)
	jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
		OP:    tarsMeta.JsonPatchAdd,
		Path:  "/spec/authentication/tokens",
		Value: tokens,
	})

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:   tarsMeta.JsonPatchRemove,
				Path: UnsafeTAccountAnnotationPath,
			})
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &tarsAppsV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsAppsV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.OldObject.Raw, oldTAccount)

	var jsonPatch tarsMeta.JsonPatch

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:   tarsMeta.JsonPatchRemove,
				Path: UnsafeTAccountAnnotationPath,
			})
		}
	}

	passwordChanged := false
	for i := 0; i < 1; i++ {
		if newTAccount.Spec.Authentication.Password != nil {
			passwordString := *newTAccount.Spec.Authentication.Password
			ok, _ := regexp.MatchString(PasswordPattern, passwordString)
			if !ok {
				err := fmt.Errorf("password should match pattern %s", PasswordPattern)
				return nil, err
			}

			bcryptPassword, _ := generateBcryptPassword(passwordString)

			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:   tarsMeta.JsonPatchRemove,
				Path: "/spec/authentication/password",
			})

			jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
				OP:    tarsMeta.JsonPatchAdd,
				Path:  "/spec/authentication/bcryptPassword",
				Value: string(bcryptPassword),
			})

			passwordChanged = true
			break
		}

		if !equality.Semantic.DeepEqual(oldTAccount.Spec.Authentication.BCryptPassword, newTAccount.Spec.Authentication.BCryptPassword) {
			passwordChanged = true
			break
		}
	}

	if passwordChanged {
		tokens := make([]tarsAppsV1beta3.TAccountAuthenticationToken, 0, 0)
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/spec/authentication/tokens",
			Value: tokens,
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}