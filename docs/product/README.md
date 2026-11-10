# Product docs

What Relay is and does, in the words of the people who use it. You don't need
to know the code to read these. How the software is built is in the
[engineering docs](../engineering/README.md).

## What goes where

| Folder             | What it holds                                                    | The question it answers                              |
| ------------------ | ---------------------------------------------------------------- | ---------------------------------------------------- |
| `personas/`        | The people who use the product                                   | Who is this for?                                     |
| `capabilities/`    | What people can do, each with its stability level and known gaps | What can someone accomplish, and how finished is it? |
| `flows/`           | How work gets done from start to finish, across capabilities     | What happens, in order, to get the job done?         |
| `domain/rules/`    | What must or can't happen                                        | Is this allowed?                                     |
| `domain/concepts/` | What a term in the domain means                                  | What exactly is this thing?                          |

A folder appears once it has a doc in it.

## Where to start

New to the product? Read the personas to see who it's for, then the flows to see
how their work gets done. Go to a capability to see what's possible today and
what's missing, and to the rules for the details.

## Deciding where something goes

- **Something people can accomplish** is a capability. A page, a button, or an
  endpoint isn't one.
- **Something that must or can't happen** goes in a rule doc, once, and
  capabilities link to it.
- **What a term means** goes in a concept doc, once, and everything else links to
  it.
- **A question nobody has settled** goes under Open decisions in its capability.
  It becomes a rule once it's decided.
- **Something missing** goes under Known gaps in its capability if someone using
  the product would notice. If only someone working on the code would, it goes in
  the engineering docs. It's never in both.

These docs name no pages, buttons, endpoints, or tables, so a redesign doesn't
make them wrong.

## Stability levels

Each capability says how far along it is:

| Level     | What it means                                                      |
| --------- | ------------------------------------------------------------------ |
| Prototype | An experiment to answer a question. It may be rewritten or removed |
| Alpha     | The main workflow is real and usable, but its shape may change     |
| Beta      | The shape has settled, and the edges are being hardened            |
| GA        | People can depend on it                                            |

## Keeping them current

A doc changes along with whatever it describes. Where a rule has an acceptance
test, that test is the specification, so changing the rule means changing the
test too.
