package lohen

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Unforeseen Strike deals no damage; all frames estimated.
var skillFrames []int

// Etched Into Bone and Soul: a thrown bullet, then a rush with 4 hits. The hits' order and
// spacing follow the normalized times in the ability data (0.086, 0.2095, 0.4074, 0.5316 of
// the rush), assuming a 60-frame rush starting 10 frames after the press; the animation
// lengths are estimated.
var (
	etchedFrames   []int
	etchedHitmarks = []int{15, 23, 34, 42}
)

const (
	c6Extend = 75 // 1.25s
)

func init() {
	skillFrames = frames.InitAbilSlice(35)
	skillFrames[action.ActionAttack] = 24
	skillFrames[action.ActionCharge] = 24
	skillFrames[action.ActionBurst] = 24
	skillFrames[action.ActionDash] = 20
	skillFrames[action.ActionJump] = 20
	skillFrames[action.ActionSwap] = 24

	etchedFrames = frames.InitAbilSlice(60)
	etchedFrames[action.ActionAttack] = 50
	etchedFrames[action.ActionCharge] = 50
	etchedFrames[action.ActionSkill] = 50
	etchedFrames[action.ActionBurst] = 50
	etchedFrames[action.ActionDash] = etchedHitmarks[3]
	etchedFrames[action.ActionJump] = etchedHitmarks[3]
	etchedFrames[action.ActionSwap] = 50
}

func (c *char) Skill(p map[string]int) (action.Info, error) {
	if c.etchedReady() {
		return c.etchedSkill(), nil
	}

	c.etched = 0
	c.joy = 0
	c.will = 0
	c.c6Mark = false
	c.joyGate = [3]int{}
	for i := range c.willGate {
		c.willGate[i] = [3]int{}
	}
	c.AddStatus(masterstrokeKey, skillDuration*60, true)
	c.highSpirits()
	c.c4OnMasterstroke()

	c.SetCDWithDelay(action.ActionSkill, skillCD, 0)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillFrames),
		AnimationLength: skillFrames[action.InvalidAction],
		CanQueueAfter:   skillFrames[action.ActionDash],
		State:           action.SkillState,
	}, nil
}

func (c *char) etchedSkill() action.Info {
	will := c.will
	c.etched++
	c.joy = 0
	if !c.c6KeepWill() {
		c.will = 0
	}
	if c.c6Mark {
		c.c6Mark = false
		c.extendMasterstroke(c6Extend)
	}
	c6Crit := c.Base.Cons >= 6
	c.Core.Log.NewEvent("lohen etched into bone and soul", glog.LogCharacterEvent, c.Index()).
		Write("will", will).
		Write("etched", c.etched)

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeSpear,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       skillEtched[c.skillLvl()] * (1 + will*skillWill),
	}
	for i, hitmark := range etchedHitmarks {
		ai := ai
		ai.Abil = fmt.Sprintf("Etched Into Bone and Soul %v", i)
		if i == len(etchedHitmarks)-1 {
			ai.HitlagFactor = 0.01
			ai.HitlagHaltFrames = 0.02 * 60
		}
		var cb info.AttackCBFunc
		if i == len(etchedHitmarks)-1 {
			cb = c.etchedFinalCB(will)
		}
		c.QueueCharTask(func() {
			snap := c.Snapshot(&ai)
			if c6Crit {
				snap.Stats[attributes.CD] += c6CD
			}
			ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 2.5)
			c.Core.QueueAttackWithSnap(ai, snap, ap, 0, c.particleCB, cb)
		}, hitmark)
	}

	return action.Info{
		Frames:          frames.NewAbilFunc(etchedFrames),
		AnimationLength: etchedFrames[action.InvalidAction],
		CanQueueAfter:   etchedFrames[action.ActionDash],
		State:           action.SkillState,
	}
}

// effects of the last hit landing: Hexerei, C2 and C6
func (c *char) etchedFinalCB(will float64) info.AttackCBFunc {
	done := false
	return func(a info.AttackCB) {
		if a.Target.Type() != info.TargettableEnemy || done {
			return
		}
		done = true
		c.hexerei(will)
		c.c2Arm()
		c.c6Trigger()
	}
}
