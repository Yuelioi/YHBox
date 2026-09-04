---
title: Data Processing Nodes
description: Logic, numbers, text, collections, JSON, and coordinates
slug: data-processing-nodes
order: 35
source_id: yotta-data-processing-nodes
---

# Data Processing Nodes

## Logic and comparison

And, Or, Not, and Select combine booleans and values. Equal, Not Equal, Greater Than, Less Than, and
their inclusive variants produce booleans that can feed Branch.

## Math

Arithmetic, Absolute, Minimum, Maximum, Clamp, rounding, and Modulo cover common calculations.
Integer and Number are distinct types; choose an explicit rounding conversion before connecting to an
integer input.

## Text

Concat, Contains, Starts With, Ends With, Index Of, Substring, Replace, Trim, case conversion, Regex
Match, and Regex Extract process strings. Prefer simple operations before introducing a regex.

## Collections and JSON

Append, Get, Slice, Length, and Contains manipulate lists. Split and Join convert text lists. Parse
JSON, JSON Path, and Stringify JSON form the common HTTP-response processing chain.

## Geometry and structures

Make Point, Offset Point, Point Distance, Region Around Point, and structure-break nodes manipulate
typed positions. Confirm coordinate units before connecting automation nodes; ratio, client pixels,
and screen pixels are not interchangeable.
