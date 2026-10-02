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
  Status: 'discovered'
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

// CandidateReviewEntry/Pool are NOT real backend structs — no endpoint
// combines Rank + GenerateExplanation + DetectPotentialBridge today. See
// the TODO in services/candidateReview.ts.
export interface CandidateReviewEntry {
  Ranked: RankedCandidate
  Explanation: CandidateExplanation
  Bridge: BridgeResult | null
  BridgeTrack: string | null
}

export interface CandidateReviewPool {
  EditionContext: string
  Entries: CandidateReviewEntry[]
}
