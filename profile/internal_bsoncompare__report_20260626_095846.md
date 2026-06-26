# Performance Profile: ./internal/bsoncompare/

| Metric | Value |
|--------|-------|
| Package | `./internal/bsoncompare/` |
| Timestamp | 20260626_095846 |
| Wall time | 3s |
| Tests | 32 |
| Slowest test | 2.53s |
| Coverage | 73.9% |
| Exit code | 0 |

## Slowest Tests

```
2.53 TestBuildIDMap_CorpusB (2.53s)
2.34 TestAssertEqual_SelfComparePasses (2.34s)
2.31 TestCompare_ContentHashSkip (2.31s)
2.29 TestReadAllUnits_CorpusB (2.29s)
2.28 TestReadAllUnits_ContentHashPopulated (2.28s)
2.28 TestCompare_NoChange (2.28s)
2.24 TestReadAllUnits_ContentHashStable (2.24s)
0.00 TestReadAllUnits_MissingPath (0.00s)
0.00 TestNormalize_VersionedArrayPrefixSkipped (0.00s)
0.00 TestNormalize_UnknownPointer (0.00s)
0.00 TestNormalize_StableIdOmitted (0.00s)
0.00 TestNormalize_SelfIDOmitted (0.00s)
0.00 TestNormalize_PointerResolved (0.00s)
0.00 TestNormalize_LayoutFieldsOmitted (0.00s)
0.00 TestNormalize_DocumentationOmitted (0.00s)
0.00 TestFormatDiff_Empty (0.00s)
0.00 TestFormatDiff_Changed (0.00s)
0.00 TestFormatDiff_Added (0.00s)
0.00 TestExpectRemoved_NotFound (0.00s)
0.00 TestExpectRemoved_Matches (0.00s)
```

## Profiles

- CPU: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__cpu_20260626_095846.pprof`
- Memory: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__mem_20260626_095846.pprof`
- Coverage: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__cover_20260626_095846.out`

## Top CPU (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: d0614fc0093a7a1be0b4db6baecc94bfd06cf530
Type: cpu
Time: 2026-06-26 09:58:46 CST
Duration: 2.53s, Total samples = 16550ms (654.29%)
Showing nodes accounting for 9370ms, 56.62% of 16550ms total
Dropped 277 nodes (cum <= 82.75ms)
Showing top 20 nodes out of 157
      flat  flat%   sum%        cum   cum%
    1020ms  6.16%  6.16%     1760ms 10.63%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).Load
     960ms  5.80% 11.96%      960ms  5.80%  hash/fnv.(*sum64a).Write
     680ms  4.11% 16.07%      680ms  4.11%  internal/runtime/syscall/linux.Syscall6
     660ms  3.99% 20.06%     2790ms 16.86%  runtime.mallocgcSmallScanNoHeader
     620ms  3.75% 23.81%      620ms  3.75%  runtime.memmove
     580ms  3.50% 27.31%      580ms  3.50%  runtime.nextFreeFast (inline)
     570ms  3.44% 30.76%     4390ms 26.53%  runtime.mallocgc
     490ms  2.96% 33.72%      580ms  3.50%  runtime.(*mspan).writeHeapBitsSmall
     450ms  2.72% 36.44%    13160ms 79.52%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
     380ms  2.30% 38.73%      420ms  2.54%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).pop
     380ms  2.30% 41.03%      380ms  2.30%  runtime.memclrNoHeapPointers
     350ms  2.11% 43.14%      550ms  3.32%  runtime.mallocgcTiny
     330ms  1.99% 45.14%    13490ms 81.51%  go.mongodb.org/mongo-driver/v2/bson.decodeTypeOrValueWithInfo
     320ms  1.93% 47.07%    13410ms 81.03%  go.mongodb.org/mongo-driver/v2/bson.(*emptyInterfaceCodec).decodeType
     320ms  1.93% 49.00%     1380ms  8.34%  go.mongodb.org/mongo-driver/v2/bson.(*typeDecoderCache).Load
     260ms  1.57% 50.57%      430ms  2.60%  go.mongodb.org/mongo-driver/v2/bson.readBytes
     260ms  1.57% 52.15%      260ms  1.57%  runtime.acquirem (inline)
     250ms  1.51% 53.66%      250ms  1.51%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).advanceFrame
     250ms  1.51% 55.17%      730ms  4.41%  reflect.Value.assignTo
     240ms  1.45% 56.62%      240ms  1.45%  indexbytebody
```

## Top Memory (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: d0614fc0093a7a1be0b4db6baecc94bfd06cf530
Type: alloc_space
Time: 2026-06-26 09:58:49 CST
Showing nodes accounting for 5205.26MB, 97.58% of 5334.56MB total
Dropped 120 nodes (cum <= 26.67MB)
Showing top 20 nodes out of 58
      flat  flat%   sum%        cum   cum%
 2349.48MB 44.04% 44.04%  4023.49MB 75.42%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
  948.52MB 17.78% 61.82%   948.52MB 17.78%  os.readFileContents
  413.35MB  7.75% 69.57%   413.35MB  7.75%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).ReadBinary
  323.01MB  6.05% 75.63%   323.01MB  6.05%  reflect.unsafe_New
  257.50MB  4.83% 80.45%   257.50MB  4.83%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readCString
  174.34MB  3.27% 83.72%   174.34MB  3.27%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readString
  145.54MB  2.73% 86.45%  3996.90MB 74.92%  go.mongodb.org/mongo-driver/v2/bson.decodeDefault
  119.50MB  2.24% 88.69%   293.85MB  5.51%  go.mongodb.org/mongo-driver/v2/bson.(*stringCodec).decodeType
     110MB  2.06% 90.75%   523.36MB  9.81%  go.mongodb.org/mongo-driver/v2/bson.binaryDecodeType
   71.52MB  1.34% 92.09%    71.52MB  1.34%  go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore.Document.Elements
   62.01MB  1.16% 93.26%    62.01MB  1.16%  reflect.unsafe_NewArray
   57.50MB  1.08% 94.33%   119.51MB  2.24%  reflect.MakeSlice
   51.50MB  0.97% 95.30%    51.50MB  0.97%  reflect.Value.extendSlice
   39.53MB  0.74% 96.04%    39.53MB  0.74%  reflect.mapassign_faststr0
   29.51MB  0.55% 96.59%   101.03MB  1.89%  go.mongodb.org/mongo-driver/v2/bson.Raw.Elements
   20.59MB  0.39% 96.98%  1003.47MB 18.81%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).buildUnitCache
   11.44MB  0.21% 97.19%   183.97MB  3.45%  github.com/mendixlabs/mxcli/internal/bsoncompare.BuildIDMap
    9.36MB  0.18% 97.37%   972.88MB 18.24%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).readMprContents
    7.56MB  0.14% 97.51%  1011.03MB 18.95%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).listUnitsByTypeV2
    3.50MB 0.066% 97.58%   172.54MB  3.23%  github.com/mendixlabs/mxcli/internal/bsoncompare.collectIDsRaw
```
