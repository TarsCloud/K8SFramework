package v1beta1

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"e2e/scaffold"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1Beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"time"
)

const BcryptHashCost = 6

func generateBcryptPassword(in string) ([]byte, error) {
	sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(in)))
	return bcrypt.GenerateFromPassword([]byte(sha1String), BcryptHashCost)
}

var _ = ginkgo.Describe("test account", func() {
	opts := &scaffold.Options{
		Name:     "default",
		SyncTime: 800 * time.Millisecond,
	}
	s := scaffold.NewScaffold(opts)

	Username := "admin"
	ResourceName := fmt.Sprintf("%x", md5.Sum([]byte(Username)))
	Password := scaffold.RandStringRunes(10)

	ginkgo.BeforeEach(func() {
		taccountLayout := &tarsV1Beta3.TAccount{
			TypeMeta: k8sMetaV1.TypeMeta{},
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Name:      ResourceName,
				Namespace: s.Namespace,
			},
			Spec: tarsV1Beta3.TAccountSpec{
				Username: Username,
				Authentication: tarsV1Beta3.TAccountAuthentication{
					Password:  &Password,
					Activated: true,
				},
				Authorization: []*tarsV1Beta3.TAccountAuthorization{},
			},
		}
		var taccount *tarsV1Beta3.TAccount
		taccount, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Create(context.TODO(), taccountLayout, k8sMetaV1.CreateOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), taccount)
		time.Sleep(s.Opts.SyncTime)
	})

	ginkgo.It("check bcrypt password ", func() {
		taccount, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), taccount)
		assert.Nil(ginkgo.GinkgoT(), taccount.Spec.Authentication.Password)
		assert.NotNil(ginkgo.GinkgoT(), taccount.Spec.Authentication.BCryptPassword)
		sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(Password)))
		assert.Nil(ginkgo.GinkgoT(), bcrypt.CompareHashAndPassword([]byte(*taccount.Spec.Authentication.BCryptPassword), []byte(sha1String)))
	})

	ginkgo.It("try update password", func() {
		taccount, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		const updateTimes = 3
		for i := 0; i < updateTimes; i++ {

			tokensPatch := tarsMeta.JsonPatch{
				{
					OP:   tarsMeta.JsonPatchAdd,
					Path: "/spec/authentication/tokens",
					Value: []tarsV1Beta3.TAccountAuthenticationToken{
						{
							Name:           "v1",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
						{
							Name:           "v2",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
						{
							Name:           "v3",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
					},
				},
			}
			tokensPatchContent, _ := json.Marshal(tokensPatch)
			taccount, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, tokensPatchContent, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), taccount)
			assert.Equal(ginkgo.GinkgoT(), len(taccount.Spec.Authentication.Tokens), 3)
			NewPassword := scaffold.RandStringRunes(10)
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  "/spec/authentication/password",
					Value: NewPassword,
				},
			}

			patchContent, _ := json.Marshal(jsonPatch)
			taccount, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), taccount)
			assert.Nil(ginkgo.GinkgoT(), taccount.Spec.Authentication.Password)
			assert.NotNil(ginkgo.GinkgoT(), taccount.Spec.Authentication.BCryptPassword)
			sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(NewPassword)))
			assert.Nil(ginkgo.GinkgoT(), bcrypt.CompareHashAndPassword([]byte(*taccount.Spec.Authentication.BCryptPassword), []byte(sha1String)))
			assert.Equal(ginkgo.GinkgoT(), len(taccount.Spec.Authentication.Tokens), 0)
		}
	})

	ginkgo.It("try update bcryptPassword", func() {
		taccount, err := tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Get(context.TODO(), ResourceName, k8sMetaV1.GetOptions{})
		assert.Nil(ginkgo.GinkgoT(), err)
		const updateTimes = 3
		for i := 0; i < updateTimes; i++ {

			tokensPatch := tarsMeta.JsonPatch{
				{
					OP:   tarsMeta.JsonPatchAdd,
					Path: "/spec/authentication/tokens",
					Value: []tarsV1Beta3.TAccountAuthenticationToken{
						{
							Name:           "v1",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
						{
							Name:           "v2",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
						{
							Name:           "v3",
							Content:        scaffold.RandStringRunes(50),
							UpdateTime:     k8sMetaV1.Now(),
							ExpirationTime: k8sMetaV1.Time{Time: time.Unix(time.Now().Unix()+180, 0)},
							Valid:          true,
						},
					},
				},
			}
			tokensPatchContent, _ := json.Marshal(tokensPatch)
			taccount, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, tokensPatchContent, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), taccount)
			assert.Equal(ginkgo.GinkgoT(), len(taccount.Spec.Authentication.Tokens), 3)

			NewPassword := scaffold.RandStringRunes(10)
			NewBcryptPassword, _ := generateBcryptPassword(NewPassword)
			jsonPatch := tarsMeta.JsonPatch{
				{
					OP:    tarsMeta.JsonPatchAdd,
					Path:  "/spec/authentication/bcryptPassword",
					Value: string(NewBcryptPassword),
				},
			}

			patchContent, _ := json.Marshal(jsonPatch)
			taccount, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TAccounts(s.Namespace).Patch(context.TODO(), ResourceName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.NotNil(ginkgo.GinkgoT(), taccount)
			assert.Nil(ginkgo.GinkgoT(), taccount.Spec.Authentication.Password)
			assert.NotNil(ginkgo.GinkgoT(), taccount.Spec.Authentication.BCryptPassword)
			sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(NewPassword)))
			assert.Nil(ginkgo.GinkgoT(), bcrypt.CompareHashAndPassword([]byte(*taccount.Spec.Authentication.BCryptPassword), []byte(sha1String)))
			assert.Equal(ginkgo.GinkgoT(), len(taccount.Spec.Authentication.Tokens), 0)
		}
	})
})
