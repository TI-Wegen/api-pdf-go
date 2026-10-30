package browser

// checkout devolve a instância atual, trocando-a se caiu ou atingiu o limite de usos.
func (p *Pool) checkout() (*instance, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current != nil && (!p.current.alive() || p.current.uses >= p.recycleAfter) {
		p.log.Info("reciclando Chromium", "vivo", p.current.alive(), "usos", p.current.uses)
		p.retire(p.current)
		p.current = nil
	}

	if p.current == nil {
		inst, err := newInstance(p.chromePath, p.log)
		if err != nil {
			return nil, err
		}
		p.current = inst
	}

	p.current.refs++
	p.current.uses++
	return p.current, nil
}

func (p *Pool) checkin(inst *instance) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inst.refs--
	if inst.retired && inst.refs == 0 {
		go inst.close()
	}
}

// retire marca a instância para fechar quando a última aba em uso terminar.
// Deve ser chamado com p.mu travado.
func (p *Pool) retire(inst *instance) {
	inst.retired = true
	if inst.refs == 0 {
		go inst.close()
	}
}
