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
		if err = ConvertData(args.Data, &config); err != nil {
			return reply
		}
		_, err = Connect(&config)
	case "disconnect":
		var names []string
		if err = ConvertData(args.Data, &names); err != nil {
			return reply
		}
		Disconnect(names...)
	case "disconnect_all":
		DisconnectAll()
	case "connection_names":
		names := ConnectionNames()
		reply.Data = map[string]any{"names": names}
	//数据源管理
	case "register_schema":
		var values []any
		if err = ConvertData(args.Data, &values); err != nil {
			return reply
		}
		err = RegisterSchema(values...)
	case "unregister_schema":
		var names []string
		if err = ConvertData(args.Data, &names); err != nil {
			return reply
		}
		err = UnregisterSchema(names...)
	case "schema_by":
		var name string
		if err = ConvertData(args.Data, &name); err != nil {
			return reply
		}
		schema := SchemaBy(name)
		reply.Data = map[string]any{"schema": schema}
	case "schema_bys":
		var names []string
		if err = ConvertData(args.Data, &names); err != nil {
			return reply
		}
		schemas := SchemaBys(names...)
		reply.Data = map[string]any{"schemas": schemas}
	// 脚本操作
	case "exec":
		var input struct {
			Query string `json:"query"`
			Args  []any  `json:"args"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if n, e := Exec(input.Query, input.Args...); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"n": n}
		}
	case "exec_batch":
		var input struct {
			Query string `json:"query"`
			Args  []any  `json:"args"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if n, e := ExecBatch(input.Query, input.Args...); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"ns": n}
		}
	case "exec_by":
		var input struct {
			ConnectionName string `json:"connection_name"`
			Query          string `json:"query"`
			Args           []any  `json:"args"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if n, e := ExecBy(input.ConnectionName, input.Query, input.Args...); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"n": n}
		}
	case "exec_by_batch":
		var input struct {
			ConnectionName string `json:"connection_name"`
			Query          string `json:"query"`
			Args           []any  `json:"args"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if n, e := ExecByBatch(input.ConnectionName, input.Query, input.Args...); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"ns": n}
		}
	case "query":
		var input struct {
			Query  string `json:"query"`
			Args   []any  `json:"args"`
			IsList bool   `json:"is_list"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		var dst any
		if input.IsList {
			dst = make([]map[string]any, 0)
		} else {
			dst = make(map[string]any)
		}
		if err = Query(&dst, input.Query, input.Args...); err != nil {
			return reply
		}
		reply.Data = map[string]any{"data": dst}
	case "query_by":
		var input struct {
			ConnectionName string `json:"connection_name"`
			Query          string `json:"query"`
			Args           []any  `json:"args"`
			IsList         bool   `json:"is_list"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		var dst any
		if input.IsList {
			dst = make([]map[string]any, 0)
		} else {
			dst = make(map[string]any)
		}
		if err = QueryBy(input.ConnectionName, &dst, input.Query, input.Args...); err != nil {
			return reply
		}
		reply.Data = map[string]any{"data": dst}
	// 模型操作
	case "model_create":
	case "model_update":
	case "model_delete":
	case "model_all":
	case "model_one":
	case "model_page":
	case "model_count":
	}
	return reply
}
