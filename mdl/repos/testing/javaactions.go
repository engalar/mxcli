package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// RecordingJavaActionRepository records every call to its methods.
type RecordingJavaActionRepository struct {
	GotIDs          []model.ID
	ListedModule    []model.ID
	ListedAll       int
	FoundQNs        []string
	GetContainerIDs []model.ID
	Updated         []*genJA.JavaAction
	Deleted         []model.ID

	GetFunc              func(model.ID) (*genJA.JavaAction, error)
	ListFunc             func(model.ID) ([]*genJA.JavaAction, error)
	ListAllFunc          func() ([]*genJA.JavaAction, error)
	FindByQualifiedNameFunc func(string) (*genJA.JavaAction, error)
	GetContainerUUIDFunc func(model.ID) (model.ID, error)
	CreateFunc           func(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateFunc           func(*genJA.JavaAction) error
	DeleteFunc           func(model.ID) error
}

var _ repos.JavaActionRepository = (*RecordingJavaActionRepository)(nil)

func (r *RecordingJavaActionRepository) Get(id model.ID) (*genJA.JavaAction, error) {
	r.GotIDs = append(r.GotIDs, id)
	if r.GetFunc != nil {
		return r.GetFunc(id)
	}
	return nil, nil
}

func (r *RecordingJavaActionRepository) List(moduleID model.ID) ([]*genJA.JavaAction, error) {
	r.ListedModule = append(r.ListedModule, moduleID)
	if r.ListFunc != nil {
		return r.ListFunc(moduleID)
	}
	return nil, nil
}

func (r *RecordingJavaActionRepository) ListAll() ([]*genJA.JavaAction, error) {
	r.ListedAll++
	if r.ListAllFunc != nil {
		return r.ListAllFunc()
	}
	return nil, nil
}

func (r *RecordingJavaActionRepository) FindByQualifiedName(qn string) (*genJA.JavaAction, error) {
	r.FoundQNs = append(r.FoundQNs, qn)
	if r.FindByQualifiedNameFunc != nil {
		return r.FindByQualifiedNameFunc(qn)
	}
	return nil, nil
}

func (r *RecordingJavaActionRepository) GetContainerUUID(id model.ID) (model.ID, error) {
	r.GetContainerIDs = append(r.GetContainerIDs, id)
	if r.GetContainerUUIDFunc != nil {
		return r.GetContainerUUIDFunc(id)
	}
	return "", nil
}

func (r *RecordingJavaActionRepository) Create(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	if r.CreateFunc != nil {
		return r.CreateFunc(parentUUID, containmentName, ja)
	}
	return nil
}

func (r *RecordingJavaActionRepository) Update(ja *genJA.JavaAction) error {
	r.Updated = append(r.Updated, ja)
	if r.UpdateFunc != nil {
		return r.UpdateFunc(ja)
	}
	return nil
}

func (r *RecordingJavaActionRepository) Delete(id model.ID) error {
	r.Deleted = append(r.Deleted, id)
	if r.DeleteFunc != nil {
		return r.DeleteFunc(id)
	}
	return nil
}
