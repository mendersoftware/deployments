package users

type UserI interface {
	GetUserID() string
	GetCustomerID() string
}

type DummyUser struct {
	Name string
}

func NewDummyUser(name string) *DummyUser {
	return &DummyUser{
		Name: name,
	}
}

func (user *DummyUser) GetUserID() string {
	return user.Name
}

func (user *DummyUser) GetCustomerID() string {
	return user.Name
}
