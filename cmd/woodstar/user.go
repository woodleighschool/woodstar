package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func userCommand() *cobra.Command {
	var databaseURL string

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage Woodstar users",
		Args:  cobra.NoArgs,
	}
	cmd.PersistentFlags().StringVar(
		&databaseURL,
		"database-url",
		"",
		"Postgres connection URL (defaults to WOODSTAR_DATABASE_URL)",
	)
	cmd.AddCommand(createUserCommand(&databaseURL))
	cmd.AddCommand(setUserPasswordCommand(&databaseURL))
	cmd.AddCommand(setUserRoleCommand(&databaseURL))
	return cmd
}

func createUserCommand(databaseURL *string) *cobra.Command {
	var email string
	var name string
	var password string
	var role string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a local user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedPassword, err := commandPassword(cmd, password)
			if err != nil {
				return err
			}
			return withUserService(cmd.Context(), *databaseURL, func(users *directory.UserService) error {
				user, err := users.Create(cmd.Context(), directory.UserCreate{
					Email:    email,
					Name:     name,
					Password: resolvedPassword,
					Role:     directory.Role(role),
				})
				if err != nil {
					return fmt.Errorf("create user: %w", err)
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "created user %s (id %d)\n", user.Email, user.ID)
				return err
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringVar(&password, "password", "", "User password (prompts when omitted)")
	cmd.Flags().StringVar(&role, "role", "", "User role: admin or viewer")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role")
	return cmd
}

func setUserPasswordCommand(databaseURL *string) *cobra.Command {
	var email string
	var password string

	cmd := &cobra.Command{
		Use:   "set-password",
		Short: "Replace a local user's password",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedPassword, err := commandPassword(cmd, password)
			if err != nil {
				return err
			}
			return withUserService(cmd.Context(), *databaseURL, func(users *directory.UserService) error {
				user, err := users.SetPasswordByEmail(cmd.Context(), email, resolvedPassword)
				if err != nil {
					return fmt.Errorf("set user password: %w", err)
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "updated password for %s\n", user.Email)
				return err
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&password, "password", "", "User password (prompts when omitted)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func setUserRoleCommand(databaseURL *string) *cobra.Command {
	var email string
	var role string

	cmd := &cobra.Command{
		Use:   "set-role",
		Short: "Set a persisted user's role",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withUserService(cmd.Context(), *databaseURL, func(users *directory.UserService) error {
				user, err := users.SetRoleByEmail(cmd.Context(), email, directory.Role(role))
				if err != nil {
					return fmt.Errorf("set user role: %w", err)
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "set role for %s to %s\n", user.Email, role)
				return err
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&role, "role", "", "User role: admin or viewer")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role")
	return cmd
}

func withUserService(
	ctx context.Context,
	databaseURL string,
	action func(*directory.UserService) error,
) error {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("WOODSTAR_DATABASE_URL"))
	}
	if databaseURL == "" {
		return errors.New("database URL is required: set WOODSTAR_DATABASE_URL or --database-url")
	}

	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	return action(directory.NewUserService(directory.NewStore(db)))
}

func commandPassword(cmd *cobra.Command, value string) (string, error) {
	if cmd.Flags().Changed("password") {
		return value, nil
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("password is required: pass --password or use an interactive terminal")
	}
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Password: "); err != nil {
		return "", err
	}
	password, err := term.ReadPassword(fd)
	_, newlineErr := fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	if newlineErr != nil {
		return "", newlineErr
	}
	return string(password), nil
}
