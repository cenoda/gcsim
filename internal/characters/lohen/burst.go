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

// The burst creates a gadget at 0.8775 of its animation state; the gadget's hits land 12, 36,
// 42, 48, 54 and 78 frames after it is created (its 0.1s counter, from the ability data).
// The animation length, and so the gadget's creation frame, is estimated.
var burstFrames []int

const (
	burstGadget = 88
	burstEnd    = 95
	burstCD     = 15 * 60
)

var burstHitmarks = []int{12, 36, 42, 48, 54, 78}

func init() {
	burstFrames = frames.InitAbilSlice(110)
	burstFrames[action.ActionAttack] = burstEnd
	burstFrames[action.ActionCharge] = burstEnd
	burstFrames[action.ActionSkill] = burstEnd
	burstFrames[action.ActionDash] = burstEnd
	burstFrames[action.ActionJump] = burstEnd
	burstFrames[action.ActionSwap] = burstEnd
}

func (c *char) Burst(p map[string]int) (action.Info, error) {
	inMasterstroke := c.masterstrokeActive()
	c6Crit := c.Base.Cons >= 6 && inMasterstroke
	c.extendMasterstroke(burstExtend)

	c.QueueCharTask(func() {
		if c.Base.Cons >= 4 && c.masterstrokeActive() {
			c.will = c.willMax()
		}
		will := c.currentWill()
		c.Core.Log.NewEvent("lohen burst consumes will to win", glog.LogCharacterEvent, c.Index()).
			Write("will", will)
		c.hexerei(will)
		if !c.c6KeepWill() {
			c.will = 0
		}
		c.c4OnBurstGadget()

		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			AttackTag:  attacks.AttackTagElementalBurst,
			ICDTag:     attacks.ICDTagElementalBurst,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeDefault,
			Element:    attributes.Cryo,
			Durability: 25,
			Mult:       burst[c.TalentLvlBurst()] * (1 + will*burstWill),
		}
		for i, hitmark := range burstHitmarks {
			ai := ai
			ai.Abil = fmt.Sprintf("Manifest Judgment %v", i)
			radius := 5.0
			if i == len(burstHitmarks)-1 {
				radius = 6.5
			}
			c.Core.Tasks.Add(func() {
				snap := c.Snapshot(&ai)
				if c6Crit {
					snap.Stats[attributes.CD] += c6CD
				}
				ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 5}, radius)
				c.Core.QueueAttackWithSnap(ai, snap, ap, 0)
			}, hitmark)
		}
	}, burstGadget)

	c.QueueCharTask(func() {
		c.c2Arm()
		c.c6Trigger()
	}, burstEnd)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(4)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstFrames),
		AnimationLength: burstFrames[action.InvalidAction],
		CanQueueAfter:   burstFrames[action.ActionDash],
		State:           action.BurstState,
	}, nil
}
