package lohen

import (
	tmpl "github.com/genshinsim/gcsim/internal/template/character"
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
)

const (
	masterstrokeKey = "lohen-masterstroke"
	// Masterstroke's remaining duration is capped at 20s by burst and C6 extensions
	masterstrokeMaxDur = 20 * 60

	joyMax       = 100
	joyPerHit    = 17 // Unforeseen Strike param 13
	joyGateDur   = 6  // 0.1s
	willMaxBase  = 100
	willSuccess  = 20 // Unforeseen Strike param 16
	willFail     = 1
	willRatio    = 10 // Unforeseen Strike param 15: hit of at least 1000% of Lohen's Base ATK
	willGateDur  = 6  // 0.1s per teammate
	etchedMaxC0  = 3
	etchedMaxC6  = 5
	particleICD  = 2 * 60
	particleKey  = "lohen-particle-icd"
	skillCD      = 18 * 60
	burstExtend  = 99 // 1.65s
	hexThreshold = 0.5
	hexBonus     = 0.4
	hexDur       = 6 * 60
	hexKey       = "lohen-hexerei"
)

type char struct {
	*tmpl.Character
	joy    float64
	will   float64
	etched int
	// frame until which each Joy source (normal attack, charged attack hit A and B) is gated
	joyGate [3]int
	// frame until which each teammate's Will to Win sources (success, fail, A1 extra) are gated
	willGate [][3]int
	c6Mark   bool
	c2Armed  bool
}

func NewChar(s *core.Core, w *character.CharWrapper, p info.CharacterProfile) error {
	c := char{}
	c.Character = tmpl.NewWithWrapper(s, w)

	c.EnergyMax = 60
	c.NormalHitNum = normalHitNum
	c.SkillCon = 3
	c.BurstCon = 5

	hex, ok := p.Params["hexerei"]
	if !ok {
		hex = 1
	}
	c.IsHexerei = (hex != 0)

	w.Character = &c

	return nil
}

func (c *char) Init() error {
	c.willGate = make([][3]int, len(c.Core.Player.Chars()))
	c.Core.Events.Subscribe(event.OnCharacterSwap, func(args ...any) {
		if args[0].(int) == c.Index() {
			c.endMasterstroke()
		}
	}, "lohen-masterstroke-swap")
	c.willInit()
	c.a4Init()
	return nil
}

func (c *char) Condition(fields []string) (any, error) {
	switch fields[0] {
	case "masterstroke":
		if c.masterstrokeActive() {
			return 1, nil
		}
		return 0, nil
	case "joy":
		return c.currentJoy(), nil
	case "will":
		return c.currentWill(), nil
	case "etched":
		if !c.masterstrokeActive() {
			return 0, nil
		}
		return c.etched, nil
	default:
		return c.Character.Condition(fields)
	}
}

func (c *char) ActionReady(a action.Action, p map[string]int) (bool, action.Failure) {
	if a == action.ActionSkill && c.etchedReady() {
		return true, action.NoFailure
	}
	return c.Character.ActionReady(a, p)
}

func (c *char) ActionStam(a action.Action, p map[string]int) float64 {
	if a == action.ActionCharge && c.masterstrokeActive() {
		return 10
	}
	return c.Character.ActionStam(a, p)
}

func (c *char) masterstrokeActive() bool {
	return c.StatusIsActive(masterstrokeKey)
}

// Joy and Will to Win reset when Masterstroke ends
func (c *char) currentJoy() float64 {
	if !c.masterstrokeActive() {
		return 0
	}
	return c.joy
}

func (c *char) currentWill() float64 {
	if !c.masterstrokeActive() {
		return 0
	}
	return c.will
}

func (c *char) endMasterstroke() {
	c.DeleteStatus(masterstrokeKey)
	c.joy = 0
	c.will = 0
	c.c6Mark = false
}

func (c *char) extendMasterstroke(dur int) {
	if !c.masterstrokeActive() {
		return
	}
	dur = min(dur, masterstrokeMaxDur-c.StatusDuration(masterstrokeKey))
	if dur > 0 {
		c.ExtendStatus(masterstrokeKey, dur)
	}
}

func (c *char) etchedMax() int {
	if c.Base.Cons >= 6 {
		return etchedMaxC6
	}
	return etchedMaxC0
}

func (c *char) etchedReady() bool {
	return c.masterstrokeActive() && c.joy >= joyMax && c.etched < c.etchedMax()
}

func (c *char) addJoy(src int) {
	if !c.masterstrokeActive() || c.etched >= c.etchedMax() {
		return
	}
	if c.Core.F < c.joyGate[src] {
		return
	}
	c.joyGate[src] = c.Core.F + joyGateDur
	c.joy = min(c.joy+joyPerHit, joyMax)
}

func (c *char) willMax() float64 {
	if c.Base.Cons >= 1 {
		return willMaxBase * c1WillMaxRatio
	}
	return willMaxBase
}

func (c *char) addWill(amt float64) {
	if c.Base.Cons >= 1 {
		amt *= c1WillEfficiency
	}
	c.will = min(c.will+amt, c.willMax())
}

// skill level including High Spirits, as an index into the talent tables
func (c *char) skillLvl() int {
	lvl := c.TalentLvlSkill()
	if c.StatusIsActive(highSpiritsKey) {
		lvl++
	}
	return min(lvl, 14)
}

// Cryo hits from normal, charged and skill attacks generate 1 particle every 2s
func (c *char) particleCB(a info.AttackCB) {
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if a.AttackEvent.Info.Element != attributes.Cryo {
		return
	}
	if c.StatusIsActive(particleKey) {
		return
	}
	c.AddStatus(particleKey, particleICD, true)
	c.Core.QueueParticle(c.Base.Key.String(), 1, attributes.Cryo, c.ParticleDelay)
}
