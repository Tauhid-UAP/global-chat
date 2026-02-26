package models

import "time"

type User struct {
	ID string
	Email string
	FirstName string
	LastName string
	PasswordHash string
	ProfileImage *string
	CreatedAt time.Time
	UpdatedAt time.Time
	IsAnonymous bool
}

func InstantiateUserByIsAnonymous(isAnonymous bool) User {
	return User {IsAnonymous: isAnonymous}
}

func InstantiateRegisteredUser() User {
	return InstantiateUserByIsAnonymous(false)
}

func InstantiateAnonymousUser() User {
	return InstantiateUserByIsAnonymous(true)
}
