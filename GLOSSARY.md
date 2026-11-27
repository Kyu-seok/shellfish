# Shellfish

A virtual pet fish that lives in the terminal. You keep it fed, and it carries on living while the app is closed.

## Language

**Fish**:
The one pet a user owns. It has a name, is never deleted unless the user releases it, and cannot die.
_Avoid_: Pet, tamagotchi, creature

**Release**:
Letting the Fish go for good, so that the user can start a new Fish. The released Fish is not kept anywhere, and the new Fish is a new Fish, not the old one with another Species.
_Avoid_: Delete, reset, abandon, rehome

**Farewell**:
The Fish swimming out of the Tank after the user confirms a Release, saying bye, before the new Fish is chosen. The Fish is released when the Farewell ends.
_Avoid_: Goodbye, exit, outro, departure animation

**Species**:
What kind of fish the Fish is, such as a clownfish or a goldfish. A Fish has one Species for life, and its Species decides its colours.
_Avoid_: Breed, type, skin, colour

**Tank**:
The space the Fish lives in and the user looks into. Food is dropped into it.
_Avoid_: Aquarium, bowl, screen

**Food**:
Something dropped into the Tank for the Fish to swim to and eat.
_Avoid_: Treat, snack

**Food Kind**:
One type of Food, such as a pellet, flake or worm. Each kind lowers Hunger by a different amount when eaten.
_Avoid_: Food type, flavor, item

**Hand**:
The user's pointer while it is over the Tank, like a hand held to the glass.
_Avoid_: Cursor, pointer, mouse

**Curiosity**:
The Fish wandering near the Hand while the Hand keeps moving. Food always comes first, and a Starving Fish has no Curiosity.
_Avoid_: Following, tracking

**Pat**:
The user touching the Fish itself, which makes it wiggle and, unless it is Starving, send up a heart.
_Avoid_: Pet, stroke, touch, click

**Hunger**:
How hungry the Fish is. It rises over real time, even while the app is closed, and eating Food lowers it.
_Avoid_: Fullness, satiety, food level

**Full**:
The Fish when its Hunger is within a small margin (FullMargin) of its lowest. Hunger rises continuously, so without the margin the Fish would stop being Full the moment after it ate. A Full Fish ignores Food.
_Avoid_: Satiated, overfed

**Mood**:
How the Fish feels, which depends entirely on its Hunger: Content, Hungry or Starving. It decides how the Fish looks and behaves.
_Avoid_: Emotion, state, health

**Age**:
How many days have passed since the Fish was named.
_Avoid_: Level, lifetime

**Detail**:
How finely the Fish is drawn: Fine or Coarse. The Fish takes up the same space in the Tank either way; Coarse only shows less of its markings and eyes. Coarse is for terminals and fonts that cannot show Fine.
_Avoid_: Size, resolution, blocks, mode, style

**Scenery**:
The fixed, non-living things that make up the Tank: seaweed, the sandy floor, shells and the water's surface. Bubbles, hearts and notes come and go, so they are not Scenery.
_Avoid_: Decor, decorations, props, objects
