package core

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/ds"
	"github.com/15mga/kiwi/util"
)

func NewService(ver string) Service {
	return Service{
		ver:           ver,
		meta:          util.M{},
		watchCodes:    make(map[kiwi.TSvc][]kiwi.TCode),
		notifyHandler: make(map[kiwi.TSvcCode][]kiwi.NotifyHandler),
	}
}

type Service struct {
	ver           string
	meta          util.M
	watchCodes    map[kiwi.TSvc][]kiwi.TCode
	notifyHandler map[kiwi.TSvcCode][]kiwi.NotifyHandler
}

func (s *Service) Svc() kiwi.TSvc {
	//TODO implement me
	panic("implement me")
}

func (s *Service) Start() {

}

func (s *Service) AfterStart() {

}

func (s *Service) Shutdown() {

}

func (s *Service) Dispose() {

}

func (s *Service) Ver() string {
	return s.ver
}

func (s *Service) Meta() util.M {
	return s.meta
}

func (s *Service) WatchNotice(msg util.IMsg, handler kiwi.NotifyHandler) {
	svc, code := kiwi.Codec().MsgToSvcCode(msg)
	slc, ok := s.watchCodes[svc]
	if ok {
		s.watchCodes[svc] = append(slc, code)
	} else {
		s.watchCodes[svc] = []kiwi.TCode{code}
	}
	sc := kiwi.MergeSvcCode(svc, code)
	handlerSlc, ok := s.notifyHandler[sc]
	if !ok {
		s.notifyHandler[sc] = []kiwi.NotifyHandler{handler}
	} else {
		s.notifyHandler[sc] = append(handlerSlc, handler)
	}
}

func (s *Service) GetWatchCodes(svc kiwi.TSvc) ([]kiwi.TCode, bool) {
	slc, ok := s.watchCodes[svc]
	return slc, ok
}

func (s *Service) OnNotice(pkt kiwi.IRcvNotice) {
	handlerSlc, ok := s.notifyHandler[kiwi.MergeSvcCode(pkt.Svc(), pkt.Code())]
	if !ok {
		return
	}
	for _, handler := range handlerSlc {
		handler(pkt)
	}
}

func (s *Service) HasNoticeWatcher(svc kiwi.TSvc, code kiwi.TCode) bool {
	_, ok := s.notifyHandler[kiwi.MergeSvcCode(svc, code)]
	return ok
}

var (
	_Services = ds.NewKSet[kiwi.TSvc, kiwi.IService](8, func(service kiwi.IService) kiwi.TSvc {
		return service.Svc()
	})
)

func RegisterSvc(services ...kiwi.IService) {
	for _, service := range services {
		_ = _Services.Add(service)
	}
}

func StartAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.Start()
	})
}

func AfterStartAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.AfterStart()
	})
}

func ShutdownAllService() {
	_Services.Iter(func(service kiwi.IService) {
		service.Shutdown()
	})
}

func GetService(svc kiwi.TSvc) (kiwi.IService, bool) {
	return _Services.Get(svc)
}

func HasService(svc kiwi.TSvc) bool {
	return _Services.Has(svc)
}

func AllService() []kiwi.IService {
	return _Services.Values()
}

func GetAllSvcMetas() map[kiwi.TSvc]kiwi.SvcMeta {
	nodeMeta := kiwi.GetNodeMeta()
	services := _Services.Values()
	m := make(map[kiwi.TSvc]kiwi.SvcMeta, len(services))
	for _, v := range services {
		m[v.Svc()] = kiwi.SvcMeta{
			Id:        nodeMeta.Id,
			Ip:        nodeMeta.Ip,
			Port:      nodeMeta.Port,
			NodeId:    nodeMeta.NodeId,
			StartTime: nodeMeta.StartTime,
			Svc:       v.Svc(),
			Ver:       v.Ver(),
		}
	}
	return m
}
