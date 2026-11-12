package kp

// Headers 503 and 597 (WinOLS 5) share the container walk named OLS5: the
// project and folder records carry raw trailers whose fields are not decoded.
// Their map and axis records are the same walk as every other header (Map),
// with the fields WinOLS added since: text extras, skip bytes, window state.
//
// These packs are read in full. They are written back only as they were
// (HeaderSealed): WinOLS 5.87 turns down a pack whose maps were replaced.
