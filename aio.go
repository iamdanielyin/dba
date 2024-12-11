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
			ConnectionName string `json:"connection_name"`
			Query          string `json:"query"`
			IsBatch        bool   `json:"is_batch"`
			Args           []any  `json:"args"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if input.IsBatch {
			if ns, e := ExecByBatch(input.ConnectionName, input.Query, input.Args...); e != nil {
				err = e
			} else {
				reply.Data = map[string]any{"ns": ns}
			}
		} else {
			if n, e := ExecBy(input.ConnectionName, input.Query, input.Args...); e != nil {
				err = e
			} else {
				reply.Data = map[string]any{"n": n}
			}
		}
	case "query":
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
		var input struct {
			ConnectionName string         `json:"connection_name"`
			ModelName      string         `json:"model_name"`
			ModelOptions   *ModelOptions  `json:"model_options"`
			Data           any            `json:"data"`
			Options        *CreateOptions `json:"options"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		// TODO 处理Tx
		if err = Model(input.ModelName, input.ModelOptions).Create(input.Data, input.Options); err != nil {
			return reply
		}
		reply.Data = map[string]any{"data": input.Data}
	case "model_update":
		var input struct {
			ConnectionName string         `json:"connection_name"`
			ModelName      string         `json:"model_name"`
			ModelOptions   *ModelOptions  `json:"model_options"`
			Filters        []any          `json:"filters"`
			Data           any            `json:"data"`
			Options        *UpdateOptions `json:"options"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		// TODO 处理Tx
		if n, e := Model(input.ModelName, input.ModelOptions).Find(input.Filters...).Update(input.Data, input.Options); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"n": n}
		}
	case "model_delete":
		var input struct {
			ConnectionName string         `json:"connection_name"`
			ModelName      string         `json:"model_name"`
			ModelOptions   *ModelOptions  `json:"model_options"`
			Filters        []any          `json:"filters"`
			Options        *DeleteOptions `json:"options"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		// TODO 处理Tx
		if n, e := Model(input.ModelName, input.ModelOptions).Find(input.Filters...).Delete(input.Options); e != nil {
			err = e
		} else {
			reply.Data = map[string]any{"n": n}
		}
	case "model_query":
		var input struct {
			ConnectionName string             `json:"connection_name"`
			ModelName      string             `json:"model_name"`
			ModelOptions   *ModelOptions      `json:"model_options"`
			Filters        []any              `json:"filters"`
			OrderBys       []string           `json:"order_bys"`
			Fields         []string           `json:"fields"`
			IsOmit         bool               `json:"is_omit"`
			Populates      []*PopulateOptions `json:"populates"`
			PageNum        int                `json:"page_num"`
			PageSize       int                `json:"page_size"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		var results []map[string]any
		res := Model(input.ModelName, input.ModelOptions).Find(input.Filters...).OrderBy(input.OrderBys...).Fields(input.Fields, input.IsOmit).PopulateBy(input.Populates...)
		if input.PageSize > 0 {
			if input.PageNum <= 0 {
				input.PageNum = 1
			}
			if totalRecords, totalPages, e := res.Paginate(input.PageNum, input.PageSize, &results); e != nil {
				err = e
				return reply
			} else {
				reply.Data = map[string]any{
					"results":       results,
					"total_records": totalRecords,
					"total_pages":   totalPages,
				}
			}
		} else {
			if err = res.All(&results); err != nil {
				return reply
			}
			reply.Data = map[string]any{
				"results":       results,
				"total_records": len(results),
			}
		}
	case "model_count":
		var input struct {
			ConnectionName string        `json:"connection_name"`
			ModelName      string        `json:"model_name"`
			ModelOptions   *ModelOptions `json:"model_options"`
			Filters        []any         `json:"filters"`
			OrderBys       []string      `json:"order_bys"`
		}
		if err = ConvertData(args.Data, &input); err != nil {
			return reply
		}
		if n, e := Model(input.ModelName, input.ModelOptions).Find(input.Filters...).OrderBy(input.OrderBys...).Count(); e != nil {
			err = e
			return reply
		} else {
			reply.Data = map[string]any{
				"n": n,
			}
		}
	}
	return reply
}
