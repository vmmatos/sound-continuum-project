// These types mirror backend Go structs (candidate.*, scoring.*) that carry
// NO JSON tags, so Go's default marshaling emits exact PascalCase keys
// (e.g. "FinalScore", not "finalScore"). Field names here match that
// real future response shape exactly, unlike spotify.ts's types (which
// mirror handler-local structs that do use snake_case json tags).

export type CandidateCategory = 'Past' | 'Present' | 'Emerging'
export type CandidateType = 'Classic' | 'Current' | 'Discovery'

export interface CandidateImage {
  URL: string
  Width: number
  Height: number
}

export interface CandidateArtist {
  SpotifyArtistID: string
  Name: string
  SpotifyURL: string
}

export interface CandidateAlbum {
  SpotifyAlbumID: string
  Name: string
  AlbumType: string
  ReleaseDate: string
  ReleaseDatePrecision: string
  Artwork: CandidateImage[]
  SpotifyURL: string
}

export interface CandidateMetadata {
  Title: string
  Artists: CandidateArtist[]
  Album: CandidateAlbum
  DurationMS: number
  Explicit: boolean
  SpotifyURL: string
  SpotifyURI: string
}

export type DiscoveryMethod =
  | 'classic_reference_artist'
  | 'current_reference_artist'
  | 'lastfm_similar_artist'
  | 'manual'
export type ProvenanceProvider = 'Spotify' | 'Last.fm' | ''

export interface SeedArtist {
  Provider: ProvenanceProvider
  ProviderArtistID: string
  Name: string
}

export interface DiscoveryProvenance {
  Method: DiscoveryMethod
  Provider: ProvenanceProvider
  Seed: SeedArtist | null
  DiscoveredArtist: SeedArtist | null
  LastFMMatch: number | null
}

export interface CandidateTrack {
  ID: string
  SpotifyTrackID: string
  Source: 'Spotify'
  Category: CandidateCategory
  Type: CandidateType
  Status: 'discovered' | 'selected' | 'under review'
  TrackTitle: string
  TrackArtist: string
  CreatedAt: string
  UpdatedAt: string
  Metadata: CandidateMetadata | null
  Provenance: DiscoveryProvenance[]
}

export interface Factors {
  Fit: number | null
  Freshness: number | null
  DiscoveryBonus: number | null
  Diversity: number | null
  PlaylistFit: number | null
  RepetitionPenalty: number | null
}

export interface Weights {
  Fit: number
  Freshness: number
  DiscoveryBonus: number
  Diversity: number
  PlaylistFit: number
  RepetitionWeight: number
}

export interface CandidateScore {
  CandidateID: string
  ModelVersion: string
  Weights: Weights
  Factors: Factors
  FinalScore: number | null
  AvailableWeight: number
}

export interface RankedCandidate {
  Rank: number
  Candidate: CandidateTrack
  Score: CandidateScore
}

export type BridgeDimension = 'mood' | 'energy' | 'texture' | 'cultural_influence'
export type BridgeSignal = 'shared_artist' | 'lastfm_artist_similarity' | 'release_era'
export type BridgeRelationship = 'matching' | 'progression' | 'contrasting' | 'unavailable'

export interface BridgeDimensionResult {
  Dimension: BridgeDimension
  Available: boolean
  Relationship: BridgeRelationship
  Score: number | null
  Evidence: boolean
}

export interface BridgeSignalResult {
  Signal: BridgeSignal
  Available: boolean
  Present: boolean
}

export interface BridgeResult {
  PotentialBridge: boolean
  EvidenceCount: number
  Dimensions: BridgeDimensionResult[]
  Signals: BridgeSignalResult[]
}

export interface CandidateExplanation {
  Text: string
  Reasons: string[]
}

// CandidateReviewEntry/Pool mirror backend/internal/review.ReviewEntry/
// ReviewPool exactly (GET /api/candidates/review). No EditionContext field —
// there is no real backend source for it (it would mean fabricating
// CurrentEditionContext/WeeklyDirection-style editorial content).
export interface CandidateReviewEntry {
  Ranked: RankedCandidate
  Explanation: CandidateExplanation
  Bridge: BridgeResult | null
  BridgeTrack: string | null
}

// WorkflowError/DiscoveryFailure mirror backend/internal/discovery's
// WorkflowError/Failure, carried through unchanged on ReviewPool (Card
// #126) so Entries == [] can be told apart from a genuinely empty pool
// (both of these nil/empty) vs. a Discovery-degraded one (e.g. a Spotify
// 429 recorded as a Failure). They're informational only — a non-empty
// Entries list is unaffected by either field.
export interface WorkflowError {
  Workflow: string
  Err: string
}

export interface DiscoveryFailure {
  Artist: string
  Stage: string
  Err: string
}

export interface CandidateReviewPool {
  Entries: CandidateReviewEntry[]
  // Go marshals a nil slice as null, not [] — both fields are nullable.
  WorkflowErrors: WorkflowError[] | null
  Failures: DiscoveryFailure[] | null
}
