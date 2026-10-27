package vesna

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

var transposeFrames []int

const (
	c1Key        = "vesna-c1"
	c1ArmedExtra = 5 * 60 // Buff_Duration_Extra
	c1Bonus      = 0.2

	c2Key = "vesna-c2"
	c2ATK = 0.4

	c6Key           = "vesna-c6"
	c6TransposeKey  = "vesna-c6-transpose"
	c6TransposeDur  = 5 * 60
	c6ComboKey      = "vesna-c6-combo"
	c6Mult          = 1.5
	c6BladeMult     = 2
	c6Elevation     = 0.2
	transposeHit    = 10
	transposeBlade  = 30
	transposeLength = 64
)

func init() {
	transposeFrames = frames.InitAbilSlice(transposeLength)
	transposeFrames[action.ActionAttack] = 56
	transposeFrames[action.ActionSkill] = 56
	transposeFrames[action.ActionBurst] = 56
	transposeFrames[action.ActionDash] = transposeBlade
	transposeFrames[action.ActionJump] = transposeBlade
	transposeFrames[action.ActionSwap] = transposeBlade
}

// C1: Armed for Action lasts 5s longer (Buff_Duration_Extra in the data; the text does not say
// it), allows 4 Lv. 3 Windborne Swords, the first of which consumes no Sword Essence, and Vesna's
// Stellar Swirl DMG is increased by 20% while it is active (Actor_ElemReactStarSwirlIce/WindDelta).
func (c *char) c1Init() {
	if c.Base.Cons < 1 {
		return
	}
	c.AddReactBonusMod(character.ReactBonusMod{
		Base: modifier.NewBase(c1Key, -1),
		Amount: func(ai info.AttackInfo) float64 {
			switch ai.AttackTag {
			case attacks.AttackTagReactionStellarSwirl, attacks.AttackTagDirectStellarSwirl:
			default:
				return 0
			}
			if !c.armed() {
				return 0
			}
			return c1Bonus
		},
	})
}

// C2: entering Armed for Action gives the maximum 6 Disciplinary Action stacks; ATK +40% while
// she has 6. Needs A1.
func (c *char) c2Init() {
	if c.Base.Cons < 2 || c.Base.Ascension < 1 {
		return
	}
	m := make([]float64, attributes.EndStatType)
	m[attributes.ATKP] = c2ATK
	c.AddStatMod(character.StatMod{
		Base:         modifier.NewBase(c2Key, -1),
		AffectedStat: attributes.ATKP,
		Amount: func() []float64 {
			if c.a1Count < a1MaxStacks {
				return nil
			}
			return m
		},
	})
}

func (c *char) c2OnArmed() {
	if c.Base.Cons < 2 {
		return
	}
	for range a1MaxStacks {
		c.a1Add()
	}
}

// C6: Vesna's Stellar Swirl DMG is elevated by 20%, on her reaction contributions and her direct
// Stellar Swirl hits.
func (c *char) c6Init() {
	if c.Base.Cons < 6 {
		return
	}
	c.Core.Events.Subscribe(event.OnApplyAttack, func(args ...any) {
		atk := args[0].(*info.AttackEvent)
		if atk.Info.ActorIndex != c.Index() || atk.Info.AttackTag != attacks.AttackTagDirectStellarSwirl {
			return
		}
		atk.Info.Elevation += c6Elevation
	}, c6Key+"-direct")
	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.ActorIndex != c.Index() || atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		atk.Info.Elevation += c6Elevation
	}, c6Key+"-reaction")
}

// C6: for 5s after a Lv. 3 Windborne Sword (Vesna_SwitchToD, sent when it starts), Normal Attack
// or the skill unleashes Windborne Sword: Transpose instead.
func (c *char) c6OnLv3() {
	if c.Base.Cons < 6 {
		return
	}
	c.AddStatus(c6TransposeKey, c6TransposeDur, true)
}

// Windborne Sword: Transpose deals 150% ATK Anemo DMG and summons a Spirit Blade dealing 200% ATK
// (Stellar Swirl DMG in Radiance: Stellar Swirl, multiplied by A1 like every Spirit Blade); in
// Armed for Action it also summons a wind pinion. It costs no Sword Essence and does not raise the
// Windborne Sword level.
func (c *char) transpose(_ map[string]int) (action.Info, error) {
	c.DeleteStatus(c6TransposeKey)
	// Vesna_DoNotResetCombo_C6: a Charged Attack can follow Transpose
	c.AddStatus(c6ComboKey, transposeLength+60, true)
	star := c.radiance()
	c.QueueCharTask(func() {
		ai := c.windborneAI("Windborne Sword: Transpose", c6Mult)
		ai.StrikeType = attacks.StrikeTypeSlash
		ai.HitlagHaltFrames = 0.04 * 60
		c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{X: 0.24, Y: 1.45}, 3), 0, 0, c.particleCB)
		c.pinion("Wind Pinion (Transpose)")
	}, transposeHit)
	c.QueueCharTask(func() {
		c.spiritBlade("Windborne Sword: Transpose Spirit Blade", star, c6BladeMult, c6BladeMult,
			combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 2}, 4), attacks.AttackTagElementalArt)
	}, transposeBlade)
	c.Core.Log.NewEvent("vesna windborne sword: transpose", glog.LogCharacterEvent, c.Index())

	return action.Info{
		Frames:          frames.NewAbilFunc(transposeFrames),
		AnimationLength: transposeFrames[action.InvalidAction],
		CanQueueAfter:   transposeFrames[action.ActionDash],
		State:           action.SkillState,
	}, nil
}
