package auth

import (
	"fmt"

	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	"golang.org/x/crypto/bcrypt"
)

func Login(username string, password string, allowSignup bool) (db.User, error) {
	user := db.User{Username: username}
	tx := db.Db.Limit(1).Find(&user)

	if tx.Error != nil {
		return db.User{}, tx.Error
	}
	if tx.RowsAffected == 0 {
		if !allowSignup {
			return db.User{}, &UserNotFoundError{Username: username}
		}
		bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return db.User{}, err
		}
		user = db.User{Username: username, Password: string(bytes)}
		err = db.Db.Create(&db.Organization{Users: []db.User{user}, ID: user.Username}).Error
		if err != nil {
			return db.User{}, &UserCreationFailedError{Username: username, Err: err}
		}
		return user, nil
	} else {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			return db.User{}, &InvalidPasswordError{Username: username}
		}
		return user, nil
	}
}

type InvalidPasswordError struct {
	Username string
}

func (r *InvalidPasswordError) Error() string {
	return fmt.Sprintf("user not found: %s", r.Username)
}

type UserCreationFailedError struct {
	Username string
	Err      error
}

func (r *UserCreationFailedError) Error() string {
	return fmt.Sprintf("failed to create user %s with error: %s", r.Username, r.Err.Error())
}

type UserNotFoundError struct {
	Username string
}

func (r *UserNotFoundError) Error() string {
	return fmt.Sprintf("user not found: %s", r.Username)
}
