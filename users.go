package main

import (
	r "github.com/dancannon/gorethink"
	"github.com/jaracil/ei"
)

type UserData struct {
	User      string                            `gorethink:"id,omitempty"`
	Pass      string                            `gorethink:"pass,omitempty"`
	Salt      string                            `gorethink:"salt,omitempty"`
	Tags      map[string]map[string]interface{} `gorethink:"tags,omitempty"`
	Templates []string                          `gorethink:"templates,omitempty"`

	// Limits
	Mask        map[string]map[string]interface{} `gorethink:"mask,omitempty"`
	MaxSessions int                               `gorethink:"maxsessions,omitempty"`
	Whitelist   []string                          `gorethink:"whitelist,omitempty"`
	Blacklist   []string                          `gorethink:"blacklist,omitempty"`
}

var Nobody *UserData = &UserData{User: "nobody", Tags: map[string]map[string]interface{}{}}

func (nc *NexusConn) handleUserReq(req *JsonRpcReq) {
	switch req.Method {
	case "user.create":
		user, err := ei.N(req.Params).M("user").Lower().F(checkRegexp, _userRegexp).F(checkLen, _userMinLen, _userMaxLen).String()
		if err != nil {
			req.Error(ErrInvalidParams, "user", nil)
			return
		}
		pass, err := ei.N(req.Params).M("pass").F(checkLen, _passwordMinLen, _passwordMaxLen).String()
		if err != nil {
			req.Error(ErrInvalidParams, "pass", nil)
			return
		}
		tags := nc.getTags(user)
		if !(ei.N(tags).M("@"+req.Method).BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		ud := UserData{User: user, Salt: safeId(16), Tags: map[string]map[string]interface{}{}, Templates: []string{}, MaxSessions: 50}
		ud.Pass, err = HashPass(pass, ud.Salt)
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		_, err = r.Table("users").Insert(&ud).RunWrite(db, r.RunOpts{Durability: "hard"})
		if err != nil {
			if r.IsConflictErr(err) {
				req.Error(ErrUserExists, "", nil)
			} else {
				req.Error(ErrInternal, "", nil)
			}
			return
		}
		req.Result(map[string]interface{}{"ok": true})

	case "user.delete":
		user, err := ei.N(req.Params).M("user").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "user", nil)
			return
		}
		tags := nc.getTags(user)
		if !(ei.N(tags).M("@"+req.Method).BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		res, err := r.Table("users").Get(user).Delete().RunWrite(db, r.RunOpts{Durability: "hard"})
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		if res.Deleted > 0 {
			req.Result(map[string]interface{}{"ok": true})
		} else {
			req.Error(ErrInvalidUser, "", nil)
		}

	case "user.setTags":
		user, err := ei.N(req.Params).M("user").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "user", nil)
			return
		}
		prefix, err := ei.N(req.Params).M("prefix").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "prefix", nil)
			return
		}
		tgs, err := ei.N(req.Params).M("tags").MapStr()
		if err != nil {
			req.Error(ErrInvalidParams, "tags", nil)
			return
		}
		tags := nc.getTags(prefix)
		if !(ei.N(tags).M("@"+req.Method).BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		res, err := r.Table("users").Get(user).Update(map[string]interface{}{"tags": map[string]interface{}{prefix: tgs}}).RunWrite(db, r.RunOpts{Durability: "hard"})
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		if res.Unchanged == 0 && res.Replaced == 0 {
			req.Error(ErrInvalidUser, "", nil)
			return
		}
		req.Result(map[string]interface{}{"ok": true})
	case "user.delTags":
		user, err := ei.N(req.Params).M("user").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "user", nil)
			return
		}
		prefix, err := ei.N(req.Params).M("prefix").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "prefix", nil)
			return
		}
		tgs, err := ei.N(req.Params).M("tags").Slice()
		if err != nil {
			req.Error(ErrInvalidParams, "tags", nil)
			return
		}
		tags := nc.getTags(prefix)
		if !(ei.N(tags).M("@"+req.Method).BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		res, err := r.Table("users").Get(user).Update(map[string]interface{}{"tags": map[string]interface{}{prefix: r.Literal(r.Row.Field("tags").Field(prefix).Without(tgs))}}).RunWrite(db, r.RunOpts{Durability: "hard"})
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		if res.Unchanged == 0 && res.Replaced == 0 {
			req.Error(ErrInvalidUser, "", nil)
			return
		}
		req.Result(map[string]interface{}{"ok": true})

	case "user.setPass":
		user, err := ei.N(req.Params).M("user").Lower().String()
		if err != nil {
			req.Error(ErrInvalidParams, "user", nil)
			return
		}
		pass, err := ei.N(req.Params).M("pass").F(checkLen, _passwordMinLen, _passwordMaxLen).String()
		if err != nil {
			req.Error(ErrInvalidParams, "pass", nil)
			return
		}
		tags := nc.getTags(user)
		if !(ei.N(tags).M("@"+req.Method).BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		salt := safeId(16)
		hp, err := HashPass(pass, salt)
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		res, err := r.Table("users").Get(user).Update(map[string]interface{}{"salt": salt, "pass": hp}).RunWrite(db, r.RunOpts{Durability: "hard"})
		if err != nil {
			req.Error(ErrInternal, "", nil)
			return
		}
		if res.Unchanged == 0 && res.Replaced == 0 {
			req.Error(ErrInvalidUser, "", nil)
			return
		}
		req.Result(map[string]interface{}{"ok": true})

	case "user.list":
		prefix := ei.N(req.Params).M("prefix").Lower().StringZ()
		limit, err := ei.N(req.Params).M("limit").Int()
		if err != nil {
			limit = 100
		}
		skip, err := ei.N(req.Params).M("skip").Int()
		if err != nil {
			skip = 0
		}
		tags := nc.getTags(prefix)
		if !(ei.N(tags).M("@user.list").BoolZ() || ei.N(tags).M("@admin").BoolZ()) {
			req.Error(ErrPermissionDenied, "", nil)
			return
		}
		term := r.Table("users").
			Between(prefix, prefix+"\uffff").
			Pluck("id", "tags", "templates", "whitelist", "blacklist", "maxsessions")

		if skip >= 0 {
			term = term.Skip(skip)
		}

		if limit >= 0 {
			term = term.Limit(limit)
		}

		cur, err := term.Map(func(row r.Term) interface{} {
			return ei.M{
				"user":        row.Field("id"),
				"tags":        row.Field("tags").Default(ei.M{}),
				"templates":   row.Field("templates").Default(ei.S{}),
				"whitelist":   row.Field("whitelist").Default(ei.S{}),
				"blacklist":   row.Field("blacklist").Default(ei.S{}),
				"maxsessions": row.Field("maxsessions").Default(10),
			}
		}).Run(db)
		if err != nil {
			req.Error(ErrInternal, err.Error(), nil)
			return
		}
		var all []interface{}
		cur.All(&all)
		req.Result(all)

	case "user.addTemplate":
		nc.userAddParam(req, "template", "templates")
	case "user.delTemplate":
		nc.userDelParam(req, "template", "templates")

	case "user.addWhitelist":
		nc.userAddParam(req, "whitelist", "whitelist")
	case "user.delWhitelist":
		nc.userDelParam(req, "whitelist", "whitelist")

	case "user.addBlacklist":
		nc.userAddParam(req, "blacklist", "blacklist")
	case "user.delBlacklist":
		nc.userDelParam(req, "blacklist", "blacklist")

	case "user.setMaxSessions":
		nc.userSetParam(req, "maxsessions", "maxsessions")

	default:
		req.Error(ErrMethodNotFound, "", nil)
	}
}

func (nc *NexusConn) userAddParam(req *JsonRpcReq, paramName, field string) {
	nc.userChangeParam(req, paramName, field, "add")
}

func (nc *NexusConn) userDelParam(req *JsonRpcReq, paramName, field string) {
	nc.userChangeParam(req, paramName, field, "del")
}

func (nc *NexusConn) userSetParam(req *JsonRpcReq, paramName, field string) {
	nc.userChangeParam(req, paramName, field, "set")
}

func (nc *NexusConn) userChangeParam(req *JsonRpcReq, paramName, field, action string) {
	user, err := ei.N(req.Params).M("user").Lower().String()
	if err != nil {
		req.Error(ErrInvalidParams, "user", nil)
		return
	}

	param, err := ei.N(req.Params).M(paramName).Lower().String()
	if err != nil {
		req.Error(ErrInvalidParams, paramName, nil)
		return
	}

	userTags := ei.N(nc.getTags(user))
	if !(userTags.M("@"+req.Method).BoolZ() || userTags.M("@admin").BoolZ()) {
		req.Error(ErrPermissionDenied, "", nil)
		return
	}

	paramTags := ei.N(nc.getTags(param))
	if !(paramTags.M("@"+req.Method).BoolZ() || paramTags.M("@admin").BoolZ()) {
		req.Error(ErrPermissionDenied, "", nil)
		return
	}
	term := r.Table("users").Get(user)

	switch action {
	case "add":
		term = term.Update(map[string]interface{}{
			field: r.Row.Field(field).Default(ei.S{}).SetInsert(param),
		})
	case "del":
		term = term.Update(map[string]interface{}{
			field: r.Row.Field(field).Default(ei.S{}).SetDifference([]string{param}),
		})
	case "set":
		term = term.Update(map[string]interface{}{field: param})
	}

	res, err := term.RunWrite(db, r.RunOpts{Durability: "hard"})
	if err != nil {
		req.Error(ErrInternal, "", nil)
		return
	}
	if res.Unchanged == 0 && res.Replaced == 0 {
		req.Error(ErrInvalidUser, "", nil)
		return
	}
	req.Result(map[string]interface{}{"ok": true})
}
