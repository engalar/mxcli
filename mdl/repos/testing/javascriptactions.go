package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// RecordingJavaScriptActionRepository records every call to its methods.
type RecordingJavaScriptActionRepository struct {
	GotIDs          []model.ID
	ListedModule    []model.ID
	ListedAll       int
	FoundQNs        []string
	GetContainerIDs []model.ID
	Updated         []*genJSA.JavaScriptAction
	Deleted         []model.ID

	GetFunc                 func(model.ID) (*genJSA.JavaScriptAction, error)
	ListFunc                func(model.ID) ([]*genJSA.JavaScriptAction, error)
	ListAllFunc             func() ([]*genJSA.JavaScriptAction, error)
	FindByQualifiedNameFunc func(string) (*genJSA.JavaScriptAction, error)
	GetContainerUUIDFunc    func(model.ID) (model.ID, error)
	CreateFunc              func(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error
	UpdateFunc              func(*genJSA.JavaScriptAction) error
	DeleteFunc              func(model.ID) error
}

var _ repos.JavaScriptActionRepository = (*RecordingJavaScriptActionRepository)(nil)

func (r *RecordingJavaScriptActionRepository) Get(id model.ID) (*genJSA.JavaScriptAction, error) {
	r.GotIDs = append(r.GotIDs, id)
	if r.GetFunc != nil {
		return r.GetFunc(id)
	}
	return nil, nil
}

func (r *RecordingJavaScriptActionRepository) List(moduleID model.ID) ([]*genJSA.JavaScriptAction, error) {
	r.ListedModule = append(r.ListedModule, moduleID)
	if r.ListFunc != nil {
		return r.ListFunc(moduleID)
	}
	return nil, nil
}

func (r *RecordingJavaScriptActionRepository) ListAll() ([]*genJSA.JavaScriptAction, error) {
	r.ListedAll++
	if r.ListAllFunc != nil {
		return r.ListAllFunc()
	}
	return nil, nil
}

func (r *RecordingJavaScriptActionRepository) FindByQualifiedName(qn string) (*genJSA.JavaScriptAction, error) {
	r.FoundQNs = append(r.FoundQNs, qn)
	if r.FindByQualifiedNameFunc != nil {
		return r.FindByQualifiedNameFunc(qn)
	}
	return nil, nil
}

func (r *RecordingJavaScriptActionRepository) GetContainerUUID(id model.ID) (model.ID, error) {
	r.GetContainerIDs = append(r.GetContainerIDs, id)
	if r.GetContainerUUIDFunc != nil {
		return r.GetContainerUUIDFunc(id)
	}
	return "", nil
}

func (r *RecordingJavaScriptActionRepository) Create(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error {
	if r.CreateFunc != nil {
		return r.CreateFunc(parentUUID, containmentName, jsa)
	}
	return nil
}

func (r *RecordingJavaScriptActionRepository) Update(jsa *genJSA.JavaScriptAction) error {
	r.Updated = append(r.Updated, jsa)
	if r.UpdateFunc != nil {
		return r.UpdateFunc(jsa)
	}
	return nil
}

func (r *RecordingJavaScriptActionRepository) Delete(id model.ID) error {
	r.Deleted = append(r.Deleted, id)
	if r.DeleteFunc != nil {
		return r.DeleteFunc(id)
	}
	return nil
}
