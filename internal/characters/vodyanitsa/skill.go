package vodyanitsa

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/enemy"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

// Timings are from Avatar_Vodyanitsa_ElementalArt and Bullet_Vodyanitsa_ElementalArt:
//   - the first hit (5.5 m around her) and Song of Ages Past start when the ability starts
//   - Song of Ages Past summons a Horn of Spring's Call every hornInterval (3s) after it starts;
//     the horn hits 0.2s after it appears (3 m around the target)
//   - healing: a pre-handler ticking every 0.2s attaches the heal handler, which heals the active
//     character every healInterval (1.5s), so the first heal is at 1.7s
//   - the party buff handler (Mark for A1, C6) is reapplied every 1s with a 2s duration
//   - Lead Vocal and Chorus (A4) are granted at 66.3% of the tap animation
//
// Animation frames are estimated: no community frame count exists. The tap skill is assumed to
// take about 0.75s like other catalyst support skills, swappable from 30 frames.
var skillFrames []int

const (
	skillHitmark   = 1
	skillCD        = 16 * 60
	hornDelay      = 12
	healFirst      = 12
	songBuffTick   = 60
	songBuffDur    = 120
	leadVocalDelay = 30

	songKey       = "song-of-ages-past"
	songBuffKey   = "vodyanitsa-song-buff"
	hydroShredKey = "vodyanitsa-hydro-shred"
	cryoShredKey  = "vodyanitsa-cryo-shred"
)

func init() {
	skillFrames = frames.InitAbilSlice(45)
	skillFrames[action.ActionAttack] = 35
	skillFrames[action.ActionCharge] = 35
	skillFrames[action.ActionBurst] = 35
	skillFrames[action.ActionDash] = 28
	skillFrames[action.ActionJump] = 28
	skillFrames[action.ActionSwap] = 30
}

func (c *char) Skill(p map[string]int) (action.Info, error) {
	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Rechitativ: Sonorous Dawn",
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Hydro,
		Durability: 25,
		Mult:       skill[c.TalentLvlSkill()],
		UseHP:      true,
	}
	c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 5.5), skillHitmark, skillHitmark, c.shredCB)

	c.Core.Tasks.Add(c.startSong, skillHitmark)
	c.Core.Tasks.Add(c.a4Grant, leadVocalDelay)

	c.SetCD(action.ActionSkill, skillCD)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillFrames),
		AnimationLength: skillFrames[action.InvalidAction],
		CanQueueAfter:   skillFrames[action.ActionDash],
		State:           action.SkillState,
	}, nil
}

func (c *char) songDuration() int {
	dur := skillDur * 60
	if c.Base.Cons >= 2 {
		dur += c2SongExtension
	}
	return dur
}

// startSong (re)starts Song of Ages Past: the skill removes the old handler and applies a new one
func (c *char) startSong() {
	src := c.Core.F
	c.songSrc = src
	dur := c.songDuration()
	c.AddStatus(songKey, dur, false)
	active := func() bool { return c.songSrc == src && c.Core.F-src < dur }

	var buff func()
	buff = func() {
		if !active() {
			return
		}
		c.AddStatus(songBuffKey, songBuffDur, false)
		c.Core.Tasks.Add(buff, songBuffTick)
	}
	buff()

	interval := int(hornInterval * 60)
	var horn func()
	horn = func() {
		if !active() {
			return
		}
		c.horn()
		c.Core.Tasks.Add(horn, interval)
	}
	c.Core.Tasks.Add(horn, interval)

	healEvery := int(healInterval * 60)
	var heal func()
	heal = func() {
		if !active() {
			return
		}
		c.heal()
		c.Core.Tasks.Add(heal, healEvery)
	}
	c.Core.Tasks.Add(heal, healFirst+healEvery)
}

func (c *char) horn() {
	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Horn of Spring's Call",
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Hydro,
		Durability: 25,
		Mult:       horn[c.TalentLvlSkill()],
		UseHP:      true,
	}
	c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 3), hornDelay, hornDelay, c.shredCB, c.particleCB)
	// C2: the bullet applies Two Voices in Black and White when it attacks, hit or not
	c.Core.Tasks.Add(c.c2Apply, hornDelay)
}

func (c *char) heal() {
	active := c.Core.Player.ActiveChar()
	amount := healFlat[c.TalentLvlSkill()] + healPct[c.TalentLvlSkill()]*c.MaxHP()
	if c.Base.Cons >= 4 {
		if active.CurrentHPRatio() >= c4HPThreshold {
			c.c4Stack()
		} else {
			amount *= 1 + c4HealBonus
		}
	}
	c.Core.Player.Heal(info.HealInfo{
		Caller:  c.Index(),
		Target:  active.Index(),
		Message: "Song of Ages Past",
		Src:     amount,
		Bonus:   c.Stat(attributes.Heal),
	})
	c.c1Apply()
}

// Hits of the skill (first hit and horns) reduce Hydro and Cryo RES for skillShredDur seconds
func (c *char) shredCB(a info.AttackCB) {
	e, ok := a.Target.(*enemy.Enemy)
	if !ok {
		return
	}
	shred := -skillShred[c.TalentLvlSkill()]
	e.AddResistMod(info.ResistMod{
		Base:  modifier.NewBaseWithHitlag(hydroShredKey, skillShredDur*60),
		Ele:   attributes.Hydro,
		Value: shred,
	})
	e.AddResistMod(info.ResistMod{
		Base:  modifier.NewBaseWithHitlag(cryoShredKey, skillShredDur*60),
		Ele:   attributes.Cryo,
		Value: shred,
	})
	if c.Core.Flags.LogDebug {
		c.Core.Log.NewEvent("vodyanitsa hydro/cryo shred", glog.LogCharacterEvent, c.Index()).
			Write("target", e.Key()).
			Write("shred", shred)
	}
}
