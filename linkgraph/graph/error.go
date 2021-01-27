package graph

import "golang.org/x/xerrors"

var (
	ErrNotFound = xerrors.New("not found")
	ErrUnknownEdgeLinks = xerrors.New("unknown source/destination edge or link")
)
