package browser

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ErrBusy indica que não houve vaga dentro do QueueTimeout.
var ErrBusy = errors.New("servidor ocupado, tente novamente")

// Pool mantém um único Chromium vivo e entrega abas com concorrência limitada.
// O navegador é reciclado após RecycleAfter usos ou se cair.
type Pool struct {
	mu           sync.Mutex
	current      *instance
	slots        chan struct{}
	recycleAfter int
	queueTimeout time.Duration
	chromePath   string
	log          *slog.Logger
}

func NewPool(maxPages, recycleAfter int, queueTimeout time.Duration, chromePath string, log *slog.Logger) *Pool {
	return &Pool{
		slots:        make(chan struct{}, maxPages),
		recycleAfter: recycleAfter,
		queueTimeout: queueTimeout,
		chromePath:   chromePath,
		log:          log,
	}
}

// Acquire espera uma vaga e devolve o contexto de uma aba nova.
// release deve ser chamado sempre: fecha a aba e libera a vaga.
func (p *Pool) Acquire(ctx context.Context) (tab context.Context, release func(), err error) {
	if err := p.waitSlot(ctx); err != nil {
		return nil, nil, err
	}

	inst, err := p.checkout()
	if err != nil {
		<-p.slots
		return nil, nil, err
	}

	tabCtx, closeTab := inst.newTab()
	release = func() {
		closeTab()
		p.checkin(inst)
		<-p.slots
	}
	return tabCtx, release, nil
}

// InFlight retorna quantas renderizações estão em andamento.
func (p *Pool) InFlight() int { return len(p.slots) }

// Close encerra o navegador atual. Chamar após o http.Server.Shutdown,
// quando não há mais abas abertas, para fechar de forma síncrona.
func (p *Pool) Close() {
	p.mu.Lock()
	inst := p.current
	p.current = nil
	idle := inst != nil && inst.refs == 0
	if inst != nil {
		inst.retired = true
	}
	p.mu.Unlock()

	if idle {
		inst.close()
	}
}

func (p *Pool) waitSlot(ctx context.Context) error {
	timer := time.NewTimer(p.queueTimeout)
	defer timer.Stop()
	select {
	case p.slots <- struct{}{}:
		return nil
	case <-timer.C:
		return ErrBusy
	case <-ctx.Done():
		return ctx.Err()
	}
}
