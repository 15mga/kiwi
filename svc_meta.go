package kiwi

type SvcMeta struct {
	Id        int64
	Ip        string
	Port      int
	NodeId    int64
	StartTime int64
	Svc       TSvc
	Ver       string
}
