# jsonpatch
Go JSON Patch library, for applying patches to objects.

Implements RFC 6902, for applying JSON patches to Go objects.

There are several existing Go JSON Patch libraries, but they either operate on maps, or bytes. If another library exists to apply patches directly to struct objects, I'm not aware of it.

Note that for structures, certain operations are impossible. Thus, this library will return errors in excess of those defined by RFC 6902. Other operations are ambiguous, and have multiple valid options. This library tries to do whatever makes the most sense, per the Principle of Least Surprise.

Specific Behavior:
- A `remove` op on a pointer field sets it to `nil`.
- A `remove` op on a value field sets it to a default-constructed object.
- An `add` op to a struct for a field which doesn't exist returns an error.
- An `add` op to a pointer field which is nil, creates a new object.
- A `move` op to or from a struct field which doesn't exist returns an error.
- A `copy` op to or from a struct field which doesn't exist returns an error.
- A `replace` op on a struct field which doesn't exist returns an error.
- A `replace` op on a pointer field which is `nil` returns an error.

# TODO
- Ops: Remove, Replace, Move, Copy, Test
- map
- interfaces, where possible (e.g. replace is possible, but add is impossible)
- benchmark, optimize
- Array types (as opposed to Slices)
- Get field name, if no tag exists (the same way `encoding/json` works)
