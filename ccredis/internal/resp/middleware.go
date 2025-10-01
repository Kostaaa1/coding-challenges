package redis

// TODO: need middleware for checking permissions, memorylimits on write commands
func middleware(handler HandlerFunc) HandlerFunc {
	return func(args []Value) Value {
		if len(args) <= 1 {
			cmd := args[0].Bulk
			return errWrongArgs(cmd)
		}
		return handler(args[1:])
	}
}
