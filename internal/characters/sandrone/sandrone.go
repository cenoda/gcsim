package sandrone

import (
	tmpl "github.com/genshinsim/gcsim/internal/template/character"
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
)

const (
	particleICD = 150 // 2.5s
	particleKey = "sandrone-particle-icd"
)

type char struct {
	*tmpl.Character
	// Fagio's Decoding Power (0-100) and Power Overdrive mode
	power     float64
	overdrive bool
	// Decoding Power lost since the last burst, 10 per Refined Tactics stack (A1)
	powerLost    float64
	powerLostSrc int
	// source frame of the current charged attack and skill hover; 0 when not active
	caSrc    int
	hoverSrc int
	// bumped by the skill to cancel a pending Decoding Power set to 100
	forceGen int
	// condensed beams counted in the current Decoding period (C2)
	c2Layers int
	a4EM     []float64
}

func NewChar(s *core.Core, w *character.CharWrapper, _ info.CharacterProfile) error {
	c := char{}
	c.Character = tmpl.NewWithWrapper(s, w)

	c.EnergyMax = 60
	c.NormalHitNum = normalHitNum
	c.NormalCon = 3
	c.BurstCon = 5

	w.Character = &c

	return nil
}

func (c *char) Init() error {
	c.stellarInit()
	c.fagioInit()
	c.a4Init()
	c.c1Init()
	return nil
}

func (c *char) Condition(fields []string) (any, error) {
	switch fields[0] {
	case "power":
		return c.power, nil
	case "overdrive":
		if c.overdrive {
			return 1, nil
		}
		return 0, nil
	case "tactics":
		return c.refinedTactics(), nil
	default:
		return c.Character.Condition(fields)
	}
}

// Condensed beams and Prism Shots (both their normal and Stellar Glimmer versions) generate 1
// particle every 2.5s. Sweeping fire, Power Overdrive shots and the burst generate none.
func (c *char) particleCB(a info.AttackCB) {
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if c.StatusIsActive(particleKey) {
		return
	}
	c.AddStatus(particleKey, particleICD, true)
	c.Core.QueueParticle(c.Base.Key.String(), 1, attributes.Cryo, c.ParticleDelay)
}
