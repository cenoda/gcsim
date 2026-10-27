package vesna

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

var (
	skillFrames []int
	wb1Frames   []int
	wb2Frames   []int
	wb3Frames   []int
)

const (
	skillHitmark = 12
	skillCD      = 18 * 60

	armedKey = "vesna-armed-for-action"

	// Avatar_Vesna_MagicPower
	essenceMax      = 90
	essenceCost     = 30
	essenceSkill    = 60
	essenceBurst    = 30
	essencePerPinon = 5

	// Lv. 1: the bullet is fired at 20% of the animation; Lv. 2: the hit and the Spirit Blade
	// gadget at 35%, the blade hits on the gadget's first think, 0.5s later
	wb1Hitmark   = 10
	wb2Hitmark   = 15
	wb2BladeWait = 30
	// Lv. 3: four Spirit Blade hits at 36%, 47%, 58% and 69% of the animation and the final hit at
	// the start of its recovery
	wb3Final = 66

	pinionTravel = 12
)

var wb3Hitmarks = []int{26, 33, 41, 49}

func init() {
	skillFrames = frames.InitAbilSlice(38)
	skillFrames[action.ActionAttack] = 30
	skillFrames[action.ActionSkill] = 30
	skillFrames[action.ActionBurst] = 30
	skillFrames[action.ActionDash] = 26
	skillFrames[action.ActionJump] = 26
	skillFrames[action.ActionSwap] = 28

	wb1Frames = frames.InitAbilSlice(42)
	wb1Frames[action.ActionAttack] = 30
	wb1Frames[action.ActionSkill] = 30
	wb1Frames[action.ActionBurst] = 30
	wb1Frames[action.ActionDash] = 28
	wb1Frames[action.ActionJump] = 28
	wb1Frames[action.ActionSwap] = 28

	wb2Frames = frames.InitAbilSlice(44)
	wb2Frames[action.ActionAttack] = 34
	wb2Frames[action.ActionSkill] = 34
	wb2Frames[action.ActionBurst] = 34
	wb2Frames[action.ActionDash] = 30
	wb2Frames[action.ActionJump] = 30
	wb2Frames[action.ActionSwap] = 30

	wb3Frames = frames.InitAbilSlice(80)
	wb3Frames[action.ActionAttack] = 70
	wb3Frames[action.ActionSkill] = 70
	wb3Frames[action.ActionBurst] = 70
	wb3Frames[action.ActionDash] = wb3Final
	wb3Frames[action.ActionJump] = wb3Final
	wb3Frames[action.ActionSwap] = wb3Final
}

func (c *char) armed() bool {
	return c.StatusIsActive(armedKey)
}

// The Art of Victory: deals AoE Anemo DMG (at the press), switches to Armed for Action and gains
// 60 Sword Essence (2 units).
func (c *char) Skill(p map[string]int) (action.Info, error) {
	if c.StatusIsActive(c6TransposeKey) {
		return c.transpose(p)
	}
	if c.armed() {
		return c.windborne()
	}

	// Vesna_ClearBuff: entering Armed for Action clears Disciplinary Action (A1)
	c.a1Clear()

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "The Art of Victory",
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Anemo,
		Durability: 25,
		Mult:       skill[c.TalentLvlSkill()],
	}
	c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 5), skillHitmark, skillHitmark, c.particleCB)

	c.startArmed()
	c.addEssence(essenceSkill, "skill")
	c.c2OnArmed()

	c.SetCDWithDelay(action.ActionSkill, skillCD, 0)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillFrames),
		AnimationLength: skillFrames[action.InvalidAction],
		CanQueueAfter:   skillFrames[action.ActionDash],
		State:           action.SkillState,
	}, nil
}

func (c *char) startArmed() {
	dur := int(armedDur * 60)
	if c.Base.Cons >= 1 {
		dur += c1ArmedExtra
	}
	c.armedSrc = c.Core.F
	c.AddStatus(armedKey, dur, true)
	c.wbLevel = 1
	c.lv3Count = 0
	c.lv3Max = 3
	c.c1Free = false
	if c.Base.Cons >= 1 {
		c.lv3Max = 4
		c.c1Free = true
	}
	src := c.armedSrc
	// Sword Essence is cleared when the mode ends, by duration or otherwise
	c.QueueCharTask(func() { c.checkArmedExpiry(src) }, dur)
}

func (c *char) checkArmedExpiry(src int) {
	if c.armedSrc != src {
		return
	}
	if c.armed() {
		// extended by hitlag
		c.QueueCharTask(func() { c.checkArmedExpiry(src) }, max(c.StatusDuration(armedKey), 1))
		return
	}
	c.endArmed()
}

func (c *char) endArmed() {
	if c.armedSrc == 0 {
		return
	}
	c.armedSrc = 0
	c.DeleteStatus(armedKey)
	c.essence = 0
	c.wbLevel = 0
	c.c1Free = false
	c.Core.Log.NewEvent("vesna armed for action ends", glog.LogCharacterEvent, c.Index())
}

func (c *char) addEssence(v float64, src string) {
	c.essence = min(c.essence+v, essenceMax)
	c.Core.Log.NewEvent("vesna sword essence", glog.LogCharacterEvent, c.Index()).
		Write("source", src).
		Write("added", v).
		Write("essence", c.essence)
}

// Windborne Sword: consumes 30 Sword Essence. Each use in the same Armed for Action raises its
// level, from Lv. 1 to Lv. 3; Lv. 2 and Lv. 3 also summon Spirit Blades. Armed for Action ends after
// the third Lv. 3 use (C1: fourth).
func (c *char) windborne() (action.Info, error) {
	level := c.wbLevel
	if level == 3 && c.c1Free {
		c.c1Free = false
	} else {
		c.essence -= essenceCost
	}
	c.Core.Log.NewEvent(fmt.Sprintf("vesna windborne sword lv. %v", level), glog.LogCharacterEvent, c.Index()).
		Write("essence_left", c.essence)

	switch level {
	case 1:
		return c.windborne1(), nil
	case 2:
		return c.windborne2(), nil
	default:
		return c.windborne3(), nil
	}
}

// onWindborne is the AttackEfff modifier of every level: level up, Disciplinary Action (A1)
func (c *char) onWindborne() {
	c.wbLevel = min(c.wbLevel+1, 3)
	c.a1Add()
}

func (c *char) windborneAI(abil string, mult float64) info.AttackInfo {
	return info.AttackInfo{
		ActorIndex:       c.Index(),
		Abil:             abil,
		AttackTag:        attacks.AttackTagElementalArt,
		ICDTag:           attacks.ICDTagVesnaMagicSword,
		ICDGroup:         attacks.ICDGroupVesnaMagicSword,
		StrikeType:       attacks.StrikeTypeDefault,
		Element:          attributes.Anemo,
		Durability:       25,
		Mult:             mult,
		HitlagFactor:     0.01,
		HitlagHaltFrames: 0.03 * 60,
	}
}

func (c *char) windborne1() action.Info {
	c.QueueCharTask(func() {
		c.onWindborne()
		ai := c.windborneAI("Windborne Sword Lv. 1", windborne1[c.TalentLvlSkill()])
		ai.StrikeType = attacks.StrikeTypePierce
		ai.HitlagHaltFrames = 0
		c.Core.QueueAttack(ai, combat.NewSingleTargetHit(c.Core.Combat.PrimaryTarget().Key()), 0, 0, c.particleCB)
	}, wb1Hitmark)

	return action.Info{
		Frames:          frames.NewAbilFunc(wb1Frames),
		AnimationLength: wb1Frames[action.InvalidAction],
		CanQueueAfter:   wb1Frames[action.ActionDash],
		State:           action.SkillState,
	}
}

func (c *char) windborne2() action.Info {
	c.QueueCharTask(func() {
		c.onWindborne()
		ai := c.windborneAI("Windborne Sword Lv. 2", windborne2[c.TalentLvlSkill()])
		c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 2.3}, 5), 0, 0, c.particleCB)

		// SkillObj_Vesna_MagicSword_Big_02 copies the Radiance state when it appears
		star := c.radiance()
		c.Core.Tasks.Add(func() {
			c.spiritBlade("Windborne Sword Lv. 2 Spirit Blade", star, blade2[c.TalentLvlSkill()], blade2SSw[c.TalentLvlSkill()],
				combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 5), attacks.AttackTagElementalArt)
		}, wb2BladeWait)
	}, wb2Hitmark)

	return action.Info{
		Frames:          frames.NewAbilFunc(wb2Frames),
		AnimationLength: wb2Frames[action.InvalidAction],
		CanQueueAfter:   wb2Frames[action.ActionDash],
		State:           action.SkillState,
	}
}

func (c *char) windborne3() action.Info {
	// the attack mode (Radiance or not) is read when the animation starts
	star := c.radiance()
	c.onWindborne()
	c.lv3Count++
	last := c.lv3Count >= c.lv3Max
	c.c6OnLv3()

	ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 1}, 6)
	for i, delay := range wb3Hitmarks {
		abil := fmt.Sprintf("Windborne Sword Lv. 3 Spirit Blade %v", i+1)
		c.QueueCharTask(func() {
			c.spiritBlade(abil, star, blade3[c.TalentLvlSkill()], blade3SSw[c.TalentLvlSkill()], ap, attacks.AttackTagElementalArt)
		}, delay)
	}
	c.QueueCharTask(func() {
		c.spiritBlade("Windborne Sword Lv. 3 Spirit Blade Final Hit", star, blade3Final[c.TalentLvlSkill()], blade3FinalSSw[c.TalentLvlSkill()], ap, attacks.AttackTagElementalArt)
		if last {
			c.endArmed()
		}
	}, wb3Final)

	return action.Info{
		Frames:          frames.NewAbilFunc(wb3Frames),
		AnimationLength: wb3Frames[action.InvalidAction],
		CanQueueAfter:   wb3Frames[action.ActionDash],
		State:           action.SkillState,
	}
}

// spiritBlade deals one Spirit Blade hit: Anemo DMG on the Vesna_MagicSword ICD, or in Radiance:
// Stellar Swirl, Anemo DMG that is Stellar Swirl DMG (StarSwirlWindDamage). Either is multiplied
// by (1 + 10% per Disciplinary Action stack) at the time of the hit (A1).
func (c *char) spiritBlade(abil string, star bool, mult, multSSw float64, ap info.AttackPattern, tag attacks.AttackTag) {
	a1 := c.a1Mult()
	if !star {
		ai := c.windborneAI(abil, mult*a1)
		ai.AttackTag = tag
		ai.StrikeType = attacks.StrikeTypeSlash
		if tag == attacks.AttackTagElementalBurst {
			ai.ICDTag = attacks.ICDTagElementalBurst
			ai.ICDGroup = attacks.ICDGroupDefault
			ai.HitlagHaltFrames = 0.05 * 60
			ai.HitlagFactor = 0.05
		}
		ai.IsDeployable = true
		c.Core.QueueAttack(ai, ap, 0, 0, c.particleCB)
		return
	}
	ai := info.AttackInfo{
		ActorIndex:       c.Index(),
		Abil:             abil + " (Stellar Swirl)",
		AttackTag:        attacks.AttackTagDirectStellarSwirl,
		ICDTag:           attacks.ICDTagNone,
		ICDGroup:         attacks.ICDGroupDefault,
		StrikeType:       attacks.StrikeTypeSlash,
		Element:          attributes.Anemo,
		Mult:             multSSw * a1,
		IgnoreDefPercent: 1,
	}
	c.Core.QueueAttack(ai, ap, 0, 0, c.particleCB)
}

// The wind pinion: summoned with every Normal, Charged and Plunging Attack in Armed for Action;
// it regenerates 5 Sword Essence when summoned and deals Anemo DMG (never Stellar Swirl DMG: the
// Radiance bullet SkillObj_Vesna_MagicSword_Small_Star runs the same ability).
func (c *char) pinion(abil string) {
	if !c.armed() {
		return
	}
	c.addEssence(essencePerPinon, "wind pinion")
	ai := c.windborneAI(abil, pinion[c.TalentLvlSkill()])
	ai.HitlagHaltFrames = 0
	ai.HitlagFactor = 0
	ai.IsDeployable = true
	c.Core.QueueAttack(ai, combat.NewSingleTargetHit(c.Core.Combat.PrimaryTarget().Key()), 0, pinionTravel, c.particleCB)
}
