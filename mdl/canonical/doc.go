// SPDX-License-Identifier: Apache-2.0

// Package canonical provides the DataType / DataTypeKind shared types used
// across the MDL executor pipeline for attribute and parameter type representation.
// The Lift/Hydrate/Persist/Codec lifecycle infrastructure (previously in entity/
// and association/ sub-packages) has been removed — those domains use the direct
// executor → backend path instead.
package canonical
