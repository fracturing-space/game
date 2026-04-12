package gamev1

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fracturing-space/game/internal/authz"
	"github.com/fracturing-space/game/internal/errs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const internalErrorMessage = "internal error"

func invalidArgument(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(codes.InvalidArgument, err.Error())
}

func internalStatus(err error) error {
	if err == nil {
		return nil
	}
	slog.Error("grpc internal failure", "error", err)
	return status.Error(codes.Internal, internalErrorMessage)
}

func mapDomainError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case authz.IsDenied(err):
		return status.Error(codes.PermissionDenied, err.Error())
	case errs.Is(err, errs.KindNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errs.Is(err, errs.KindAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errs.Is(err, errs.KindConflict):
		return status.Error(codes.Aborted, err.Error())
	case errs.Is(err, errs.KindFailedPrecondition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errs.Is(err, errs.KindInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case isPlayReadinessRejection(err):
		return mapPlayReadinessError(err)
	default:
		slog.Error("grpc unexpected domain failure", "error", err)
		return status.Error(codes.Internal, internalErrorMessage)
	}
}
