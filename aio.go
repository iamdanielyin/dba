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
	var err error
	defer func() {
		if err != nil {
			reply.Code = -1
			reply.Msg = err.Error()
		}
	}()
	// TODO 实现所有action
	switch args.Action {
	// 连接管理
	case "connect":
		var config ConnectConfig
		if err = Map2Struct(args.Data, &config); err != nil {
			return reply
		}
		_, err = Connect(&config)
	case "disconnect":
	case "disconnect_all":
	case "connection_names":
	//数据源管理
	case "register_schema":
	case "schema_by":
	case "schema_bys":
	// 模型操作
	case "model_create":
	case "model_update":
	case "model_delete":
	case "model_all":
	case "model_one":
	case "model_page":
	case "model_count":
	// 脚本操作
	case "exec":
	case "exec_batch":
	case "exec_by":
	case "exec_by_batch":
	case "query":
	case "query_by":
	}
	return reply
}
