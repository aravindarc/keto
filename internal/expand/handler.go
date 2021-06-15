package expand

import (
	"context"
	"github.com/pkg/errors"
	"net/http"
	"strconv"

	"github.com/ory/herodot"

	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"

	"google.golang.org/grpc"

	"github.com/ory/keto/internal/relationtuple"

	"github.com/julienschmidt/httprouter"

	"github.com/ory/keto/internal/x"
)

type (
	handlerDependencies interface {
		EngineProvider
		x.LoggerProvider
		x.WriterProvider
	}
	handler struct {
		d handlerDependencies
	}
)

var _ acl.ExpandServiceServer = (*handler)(nil)

const RouteBase = "/expand"

func NewHandler(d handlerDependencies) *handler {
	return &handler{d: d}
}

func (h *handler) RegisterReadRoutes(r *x.ReadRouter) {
	r.GET(RouteBase, h.getExpand)
}

func (h *handler) RegisterWriteRoutes(_ *x.WriteRouter) {}

func (h *handler) RegisterReadGRPC(s *grpc.Server) {
	acl.RegisterExpandServiceServer(s, h)
}

func (h *handler) RegisterWriteGRPC(s *grpc.Server) {}

// swagger:parameters getExpand
// nolint:deadcode,unused
type getExpandRequest struct {
	Depth int `json:"max-depth"`
}

// swagger:route GET /expand read getExpand
//
// Expand a Relation Tuple
//
// Use this endpoint to expand a relation tuple.
//
//     Consumes:
//     -  application/x-www-form-urlencoded
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       200: expandTree
//       400: genericError
//       404: genericError
//       500: genericError
func (h *handler) getExpand(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	depth, err := strconv.ParseInt(r.URL.Query().Get("max-depth"), 0, 0)
	if err != nil {
		h.d.Writer().WriteError(w, r, herodot.ErrBadRequest.WithError(err.Error()))
		return
	}

	subjectTuple := (&relationtuple.SubjectSet{}).FromURLQuery(r.URL.Query())

	// checking required parameters
	if subjectTuple.Namespace == "" {
		h.d.Writer().WriteError(w, r, herodot.ErrBadRequest.WithReason("namespace has to be specified"))
		return
	}

	if subjectTuple.Object == "" {
		h.d.Writer().WriteError(w, r, herodot.ErrBadRequest.WithReason("object has to be specified"))
		return
	}

	if subjectTuple.Relation == "" {
		h.d.Writer().WriteError(w, r, herodot.ErrBadRequest.WithReason("relation has to be specified"))
		return
	}

	res, err := h.d.ExpandEngine().BuildTree(r.Context(), subjectTuple, int(depth))
	if err != nil {
		h.d.Writer().WriteError(w, r, err)
		return
	}

	h.d.Writer().Write(w, r, res)
}

func (h *handler) Expand(ctx context.Context, req *acl.ExpandRequest) (*acl.ExpandResponse, error) {
	sub, err := relationtuple.SubjectFromProto(req.Subject)
	if err != nil {
		return nil, err
	}

	// checking required parameters
	if sub.(*relationtuple.SubjectSet).Namespace == "" {
		return nil, errors.New("namespace is required")
	}

	if sub.(*relationtuple.SubjectSet).Object == "" {
		return nil, errors.New("object is required")
	}

	if sub.(*relationtuple.SubjectSet).Relation == "" {
		return nil, errors.New("relation is required")
	}

	tree, err := h.d.ExpandEngine().BuildTree(ctx, sub, int(req.MaxDepth))
	if err != nil {
		return nil, err
	}

	return &acl.ExpandResponse{Tree: tree.ToProto()}, nil
}
