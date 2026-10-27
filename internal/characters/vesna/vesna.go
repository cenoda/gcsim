package vesna

import (
	tmpl "github.com/genshinsim/gcsim/internal/template/character"
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/keys"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	particleICD = 150 // Vesna_DropElemBall: 2.5s
	particleKey = "vesna-particle-icd"
)

type char struct {
	*tmpl.Character
	// source frame of the current Armed for Action; 0 when it is not active
	armedSrc int
	// Sword Essence, 0 to 90 (Avatar_Vesna_MagicPower)
	essence float64
	// level of the next Windborne Sword (1 to 3) and the number of Lv. 3 uses in this Armed for Action
	wbLevel  int
	lv3Count int
	lv3Max   int
	// C1: the first Lv. 3 Windborne Sword of an Armed for Action consumes no Sword Essence
	c1Free bool
	// Disciplinary Action (A1): stack count and the expiry frame of every instance
	a1Count  int
	a1Expiry []int
}

func NewChar(s *core.Core, w *character.CharWrapper, _ info.CharacterProfile) error {
	c := char{}
	c.Character = tmpl.NewWithWrapper(s, w)

	c.EnergyMax = 60
	c.NormalHitNum = normalHitNum
	c.SkillCon = 3
	c.BurstCon = 5

	w.Character = &c

	return nil
}

func (c *char) Init() error {
	c.stellarInit()
	c.a4Init()
	c.c1Init()
	c.c2Init()
	c.c6Init()

	c.Core.Events.Subscribe(event.OnCharacterSwap, func(args ...any) {
		if args[0].(int) != c.Index() {
			return
		}
		// leaving the field ends Armed for Action, the C6 window and every Disciplinary Action stack
		c.endArmed()
		c.DeleteStatus(c6TransposeKey)
		c.a1Clear()
	}, "vesna-swap")
	return nil
}

func (c *char) Condition(fields []string) (any, error) {
	switch fields[0] {
	case "armed":
		return c.armed(), nil
	case "essence":
		return int(c.essence), nil
	case "windborne-level":
		if !c.armed() {
			return 0, nil
		}
		return c.wbLevel, nil
	case "lv3-count":
		return c.lv3Count, nil
	case "disciplinary-action":
		return c.a1Stacks(), nil
	case "radiance":
		return c.radiance(), nil
	default:
		return c.Character.Condition(fields)
	}
}

// radiance reports whether Vesna is in Radiance: Stellar Swirl (_IS_IN_STARMODE_SWIRL), which the
// engine gives every party member for 8s after a Stellar Swirl
func (c *char) radiance() bool {
	return c.StatusIsActive(reactable.RadianceSwirlKey)
}

// While Armed for Action is active the Elemental Skill button becomes Windborne Sword, which needs
// 30 Sword Essence (energyMin of skills 11433, 11434, 11436); with C1 the first Lv. 3 use is free.
// During the C6 window both buttons unleash Windborne Sword: Transpose.
func (c *char) ActionReady(a action.Action, p map[string]int) (bool, action.Failure) {
	if a == action.ActionSkill {
		if c.StatusIsActive(c6TransposeKey) {
			return true, action.NoFailure
		}
		if c.armed() {
			if c.essence >= essenceCost || (c.wbLevel == 3 && c.c1Free) {
				return true, action.NoFailure
			}
			return false, action.SkillCD
		}
	}
	return c.Character.ActionReady(a, p)
}

func (c *char) NextQueueItemIsValid(k keys.Char, a action.Action, p map[string]int) error {
	if a == action.ActionCharge && c.StatusIsActive(c6ComboKey) {
		return nil
	}
	return c.Character.NextQueueItemIsValid(k, a, p)
}

// 1 Anemo particle every 2.5s from her skill, Windborne Sword, wind pinion, Spirit Blade and burst
// hits (Vesna_Elemball_Drop: configID 2021, baseEnergy 3)
func (c *char) particleCB(a info.AttackCB) {
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if c.StatusIsActive(particleKey) {
		return
	}
	c.AddStatus(particleKey, particleICD, false)
	c.Core.QueueParticle(c.Base.Key.String(), 1, attributes.Anemo, c.ParticleDelay)
}
