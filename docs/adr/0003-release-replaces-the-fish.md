# Releasing replaces the Fish, and keeps no archive

A user who wants a different Fish can now release theirs: it is let go for good and the user names a new Fish, of any Species, from the start, as on a first launch. We chose this over letting a Fish change its Species, because a Species is part of who a Fish is (ADR 0002 stands): a goldfish that turns into a koi is not the same pet, and its Age and Hunger would mean nothing carried over. The released Fish is not kept anywhere: no archive, backup or list of past Fish. Shellfish stays one Fish and one save file, and the old Fish stays in that file until the new one is named, so backing out at any point before then loses nothing. Once the new Fish is named, the release cannot be undone.

## Considered Options

- **Change the Fish's Species in place:** keeps the name and Age, but breaks "one Species for life", and an old Fish in a new body is a different pet that pretends to be the same one.
- **Keep released Fish in an archive:** lets a user bring one back, but adds a second kind of save, a way to browse it, and a Fish that is neither living nor gone.
- **Delete the save file by hand:** works today, but is hidden, easy to get wrong, and means leaving the app.
