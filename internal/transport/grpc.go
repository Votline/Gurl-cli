package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/Votline/Gurl-cli/internal/config"
	"go.uber.org/zap"

	"github.com/Votline/Gurlf"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	refl "github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

func (t *Transport) DoGRPC(c *config.GRPCConfig, resObj *Result) error {
	const op = "transport.DoGRPC"

	var res Result
	var err error
	ic := c.GetIgnrCrt() != nil
	if bytes.Equal(c.GetIgnrCrt(), []byte("false")) {
		ic = false
	}

	if len(c.ProtoPath) == 0 {
		res, err = t.doReflect(c, ic)
	} else {
		res, err = t.doProto(c, ic)
	}

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	resObj.Raw = res.Raw
	resObj.Info = res.Info
	return nil
}

func (t *Transport) doReflect(c *config.GRPCConfig, ic bool) (Result, error) {
	const op = "transport.doReflect"

	target := unsafe.String(unsafe.SliceData(c.Target), len(c.Target))
	endpoint := unsafe.String(unsafe.SliceData(c.Endpoint), len(c.Endpoint))
	dialOpts := unsafe.String(unsafe.SliceData(c.DialOpts), len(c.DialOpts))

	conn, err := t.getConn(target, ic, dialOpts)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", op, err)
	}
	defer conn.Close()

	if ic {
		t.log.Warn("Applied InsecureSkipVerify",
			zap.String("op", op),
			zap.String("target", target))
	}

	ctx := getContext(c.Metadata)

	rc := refl.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	svcName, mtName := parseEndpoint(endpoint)

	svc, err := rc.ResolveService(svcName)
	if err != nil {
		return Result{}, fmt.Errorf("%s: resolve service: %w", op, err)
	}
	mthd := svc.FindMethodByName(mtName)
	if mthd == nil {
		return Result{}, fmt.Errorf("%s: find method: no method find", op)
	}

	msg := dynamic.NewMessage(mthd.GetInputType())
	if len(c.Data) > 0 {
		if err := msg.UnmarshalJSON(c.Data); err != nil {
			return Result{}, fmt.Errorf("%s: unmarshal body: %w", op, err)
		}
	}

	stub := grpcdynamic.NewStub(conn)
	rpcRes, err := stub.InvokeRpc(ctx, mthd, msg)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return Result{Raw: []byte(err.Error())}, nil
		}
		return Result{Raw: []byte(st.Message()), Info: Status{
			Code:       int(st.Code()),
			Message:    st.Message(),
			ConfigType: "grpc",
		}}, nil
	}

	dMsg, ok := rpcRes.(*dynamic.Message)
	if !ok {
		return Result{}, fmt.Errorf("%s: type assert response: invalid response type", op)
	}

	return Result{Raw: parseMsg(dMsg), Info: Status{
		Code:       0,
		Message:    "0 OK",
		ConfigType: "grpc",
	}}, nil
}

func (t *Transport) doProto(c *config.GRPCConfig, ic bool) (Result, error) {
	const op = "transport.doProto"

	target := unsafe.String(unsafe.SliceData(c.Target), len(c.Target))
	endpoint := unsafe.String(unsafe.SliceData(c.Endpoint), len(c.Endpoint))
	protoPath := unsafe.String(unsafe.SliceData(c.ProtoPath), len(c.ProtoPath))
	importPaths := unsafe.String(unsafe.SliceData(c.ImportPaths), len(c.ImportPaths))
	dialOpts := unsafe.String(unsafe.SliceData(c.DialOpts), len(c.DialOpts))

	conn, err := t.getConn(target, ic, dialOpts)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", op, err)
	}
	defer conn.Close()

	if ic {
		t.log.Warn("Applied InsecureSkipVerify",
			zap.String("op", op),
			zap.String("target", target))
	}

	ctx := getContext(c.Metadata)

	protoDir := filepath.Dir(protoPath)
	protoFile := filepath.Base(protoPath)

	allImportPaths := getDependencyPaths(importPaths)
	allImportPaths = append(allImportPaths, protoDir)
	allImportPaths = append(allImportPaths, ".")

	parser := protoparse.Parser{
		ImportPaths: allImportPaths,
	}
	fds, err := parser.ParseFiles(protoFile)
	if err != nil {
		return Result{}, fmt.Errorf("%s: parse file: %w", op, err)
	}

	svcName, mtName := parseEndpoint(endpoint)

	svc := getFilesSvc(svcName, fds)
	if svc == nil {
		return Result{}, fmt.Errorf("%s: find service: no service find", op)
	}
	mthd := svc.FindMethodByName(mtName)
	if mthd == nil {
		return Result{}, fmt.Errorf("%s: find method: no method find", op)
	}

	msg := dynamic.NewMessage(mthd.GetInputType())
	if len(c.Data) > 0 {
		if err := msg.UnmarshalJSON(c.Data); err != nil {
			return Result{}, fmt.Errorf("%s: unmarshal body: %w", op, err)
		}
	}

	stub := grpcdynamic.NewStub(conn)
	rpcRes, err := stub.InvokeRpc(ctx, mthd, msg)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return Result{Raw: []byte(err.Error())}, nil
		}
		return Result{Raw: []byte(st.Message()), Info: Status{
			Code:       int(st.Code()),
			Message:    st.Message(),
			ConfigType: "grpc",
		}}, nil
	}

	dMsg, ok := rpcRes.(*dynamic.Message)
	if !ok {
		return Result{}, fmt.Errorf("%s: type assert response: invalid response type", op)
	}

	return Result{Raw: parseMsg(dMsg), Info: Status{
		Code:       0,
		Message:    "0 OK",
		ConfigType: "grpc",
	}}, nil
}

func (t *Transport) getDialOpts(rawOpts string, ic bool, yield func(grpc.DialOption)) error {
	const op = "transport.getDialOpts"

	if len(rawOpts) == 0 {
		if ic {
			yield(grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		} else {
			yield(grpc.WithInsecure())
		}
		return nil
	}

	cfgOpts := strings.SplitSeq(rawOpts, ";")
	for opt := range cfgOpts {
		switch opt {
		case "insecure":
			yield(grpc.WithInsecure())
		case "tls":
			if ic {
				t.log.Warn("InsecureSkipVerify is true",
					zap.String("op", op))
				yield(grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: ic,
				})))
			}
		case "tls_insecure":
			yield(grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
		case "block":
			yield(grpc.WithBlock())
		case "timeout":
			yield(grpc.WithTimeout(time.Second))
		default:
			return fmt.Errorf("%s: unknown dial option %q", op, opt)
		}
	}

	return nil
}

func (t *Transport) getConn(target string, ic bool, dialOpts string) (*grpc.ClientConn, error) {
	const op = "transport.getConn"

	opts := make([]grpc.DialOption, 0, strings.Count(dialOpts, ";"))
	if err := t.getDialOpts(dialOpts, ic, func(opt grpc.DialOption) {
		opts = append(opts, opt)
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("%s: dial: %w", op, err)
	}

	return conn, nil
}

func getContext(cfgMd []byte) context.Context {
	const op = "transport.getContext"

	ctx := context.Background()
	if len(cfgMd) > 0 {
		md := make(map[string]string)
		sData, err := gurlf.Scan(cfgMd)
		if err != nil {
			// TODO: log warn
		}

		for _, d := range sData {
			if len(d.RawData) == 0 {
				continue
			}

			for _, ent := range d.Entries {
				if ent.ValEnd == 0 {
					continue
				}

				key := unsafe.String(unsafe.SliceData(d.RawData[ent.KeyStart:ent.KeyEnd]), ent.KeyEnd-ent.KeyStart)
				val := d.RawData[ent.ValStart:ent.ValEnd]

				md[key] = string(val)
			}
		}
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(md))
	}

	return ctx
}

func getDependencyPaths(protoPath string) []string {
	const op = "transport.getDependencyPaths"

	var paths []string
	if strings.IndexByte(protoPath, '\n') == -1 {
		return []string{protoPath}
	}
	parts := strings.SplitSeq(protoPath, "\n")

	for path := range parts {
		if len(path) == 0 {
			continue
		}

		paths = append(paths, path)
	}

	return paths
}

func parseEndpoint(endp string) (string, string) {
	const op = "transport.parseEndpoint"

	if len(endp) == 0 {
		return "", ""
	}

	idx := strings.IndexByte(endp, '/')
	if idx == -1 {
		return "", ""
	}

	return endp[:idx], endp[idx+1:]
}

func getFilesSvc(sName string, fds []*desc.FileDescriptor) *desc.ServiceDescriptor {
	const op = "transport.getFields"

	var svc *desc.ServiceDescriptor
	for _, fd := range fds {
		svc = fd.FindService(sName)
		if svc != nil {
			return svc
		}
	}

	return nil
}

func parseMsg(msg *dynamic.Message) []byte {
	if jsonBytes, err := msg.MarshalJSON(); err == nil {
		return jsonBytes
	}

	if textBytes, err := msg.MarshalText(); err == nil {
		return textBytes
	}

	if rawBytes, err := msg.Marshal(); err == nil {
		return rawBytes
	}

	return nil
}
