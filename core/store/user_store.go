package store

import (
	"context"
	"database/sql"
	"github.com/Tauhid-UAP/global-chat/core/models"
)

var DB *sql.DB

func CreateUser(ctx context.Context, user models.User) error {
	_, err := DB.ExecContext(
		ctx,
		`INSERT INTO users VALUES ($1,$2,$3,$4,$5,$6,now(),now())`, user.ID, user.Email, user.FirstName, user.LastName, user.PasswordHash, user.ProfileImage)

	return err
}

func GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	u := models.InstantiateRegisteredUser()
	err := DB.QueryRowContext(ctx, `SELECT id,email,first_name,last_name,password_hash,profile_image,created_at,updated_at FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &u.ProfileImage, &u.CreatedAt, &u.UpdatedAt)

	return u, err
}

func GetUserByID(ctx context.Context, id string) (models.User, error) {
	u := models.InstantiateRegisteredUser()
	err := DB.QueryRowContext(ctx, `SELECT id,email,first_name,last_name,password_hash,profile_image,created_at,updated_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &u.ProfileImage, &u.CreatedAt, &u.UpdatedAt)

	return u, err
}

func UpdateUser(ctx context.Context, u models.User) error {
	_, err := DB.ExecContext(ctx, `UPDATE users SET first_name=$1,last_name=$2,profile_image=$3,updated_at=now() WHERE id=$4`, u.FirstName, u.LastName, u.ProfileImage, u.ID)

	return err
}
