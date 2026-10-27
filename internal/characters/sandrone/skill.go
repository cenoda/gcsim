package sandrone

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Sandrone boards the hovermech for up to 6s. On boarding, Decoding Power drops by 70, then by 10
// every 0.1s while she hovers (MarionetteNew_ElementalArt_ReduceFrost), and a Prismatic Resonance
// Cannon fires 2 Prism Shots: the first 0.25s after the press, the second 0.3s later, whose
// Radiance state is read at 0.3s (Avatar_MarionetteNew_Robot). The shots travel at 25 m/s; 12
// frames of travel are assumed.
// Frames are estimated: the earliest cancel into a charged attack (EN0 C / ED C in the KQM quick
// guide) is set at 22 frames, after the third 0.1s drain, so that a cancel at it repairs Fagio
// fully from 100 Decoding Power, which the guide says a too early cancel does not.
var skillFrames []int

const (
	skillCD         = 4 * 60
	hoverDur        = 6 * 60
	hoverDrainFirst = 70
	hoverDrain      = 10
	hoverTick       = 6
	prismFirst      = 15
	prismSecond     = 33
	prismRadiance   = 18
	prismTravel     = 12
	a1PrismMult     = 4
	a1PowerMin      = 50
)

func init() {
	skillFrames = frames.InitAbilSlice(40)
	skillFrames[action.ActionAttack] = 22
	skillFrames[action.ActionCharge] = 22
	skillFrames[action.ActionBurst] = 22
	skillFrames[action.ActionDash] = 20
	skillFrames[action.ActionJump] = 20
	skillFrames[action.ActionSwap] = 22
	skillFrames[action.ActionWalk] = 30
}

func (c *char) Skill(p map[string]int) (action.Info, error) {
	// A1: the second Prism Shot deals 400% if Decoding Power was at least 50 when boarding
	a1 := c.Base.Ascension >= 1 && c.power >= a1PowerMin

	c.forceGen++
	src := c.Core.F + 1
	c.hoverSrc = src
	c.reducePower(hoverDrainFirst)
	var drain func()
	drain = func() {
		if c.hoverSrc != src {
			return
		}
		c.reducePower(hoverDrain)
		c.Core.Tasks.Add(drain, hoverTick)
	}
	c.Core.Tasks.Add(drain, hoverTick)
	endHover := func() {
		if c.hoverSrc == src {
			c.hoverSrc = 0
		}
	}
	c.Core.Tasks.Add(endHover, hoverDur)

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Prism Shot 1",
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       prism[c.TalentLvlSkill()],
	}
	c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 0.5), prismFirst, prismFirst+prismTravel, c.particleCB)

	c.Core.Tasks.Add(func() {
		ai := ai
		ai.Abil = "Prism Shot 2"
		radiance := c.getRadiance()
		if radiance != radianceNone {
			c.stellarInfo(&ai, ai.Abil, radiance, prismSSC, prismSSw, c.TalentLvlSkill())
			if a1 {
				ai.Mult *= a1PrismMult
			}
		}
		c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 0.5), prismSecond-prismRadiance, prismSecond-prismRadiance+prismTravel, c.particleCB, c.c4CB)
	}, prismRadiance)

	c.SetCD(action.ActionSkill, skillCD)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillFrames),
		AnimationLength: skillFrames[action.InvalidAction],
		CanQueueAfter:   skillFrames[action.ActionDash],
		State:           action.SkillState,
		OnRemoved:       func(next action.AnimationState) { endHover() },
	}, nil
}
