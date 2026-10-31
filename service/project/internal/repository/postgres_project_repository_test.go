package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"taskmanager/service/project/internal/model"
)

func TestMapProjectErrorConstraintViolations(t *testing.T) {
	cases := []struct {
		name string
		err  *pgconn.PgError
		kind model.ErrorKind
	}{
		{"màu sai định dạng", &pgconn.PgError{Code: "23514", ConstraintName: "ck_project_projects_color_format"}, model.ErrorKindColorInvalid},
		{"tên quá dài", &pgconn.PgError{Code: "22001"}, model.ErrorKindNameTooLong},
	}
	for _, tc := range cases {
		var de *model.Error
		if got := mapProjectError(tc.err); !errors.As(got, &de) || de.Kind != tc.kind {
			t.Fatalf("%s: phải ra kind %q, nhận %v", tc.name, tc.kind, got)
		}
	}

	blank := &pgconn.PgError{Code: "23514", ConstraintName: "ck_project_projects_name_not_blank"}
	if got := mapProjectError(blank); !errors.Is(got, model.ErrInvalid) {
		t.Fatalf("tên rỗng phải ra ErrInvalid, nhận %v", got)
	}
}
