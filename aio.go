package dba

type AioArgs struct {
	Action string         `msgpack:"action"`
	Data   map[string]any `msgpack:"data"`
}

type AioReply struct {
	Code int            `msgpack:"code"`
	Msg  string         `msgpack:"msg"`
	Data map[string]any `msgpack:"data"`
	Rid  string         `msgpack:"rid"`
}

func HandleAio(args *AioArgs) *AioReply {
	reply := &AioReply{
		Code: 0,
		Rid:  NewUUIDToken(),
	}
	// TODO 实现所有action
	switch args.Action {
	case "connect":
	case "disconnect":
	case "disconnect_all":
	case "session":
	case "connection_names":
	case "register_schema":
	case "schema_by":
	case "schema_bys":
	case "model_create":
	case "model_update":
	case "model_delete":
	case "model_all":
	case "model_one":
	case "model_page":
	case "model_count":
	case "exec":
	case "exec_batch":
	case "exec_by":
	case "exec_by_batch":
	case "query":
	case "query_by":
	}
	return reply
}
