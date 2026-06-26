# Performance Profile: ./internal/bsoncompare

| Metric | Value |
|--------|-------|
| Package | `./internal/bsoncompare` |
| Timestamp | 20260626_094215 |
| Wall time | 7s |
| Tests | 29 |
| Slowest test | 5.68s |
| Coverage | 88.7% |
| Exit code | 0 |

## Slowest Tests

```
5.68 TestAssertEqual_SelfComparePasses (5.68s)
5.66 TestCompare_NoChange (5.66s)
3.33 TestBuildIDMap_CorpusB (3.33s)
3.16 TestReadAllUnits_CorpusB (3.16s)
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
0.00 TestExpectNoOtherChanges_ExtraUnit (0.00s)
0.00 TestExpectChanged_NotFound (0.00s)
0.00 TestExpectChanged_Matches (0.00s)
```

## Profiles

- CPU: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare_cpu_20260626_094215.pprof`
- Memory: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare_mem_20260626_094215.pprof`
- Coverage: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare_cover_20260626_094215.out`

## Top CPU (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: 9fd7dd0ec04791be6794639216676a1f1a946da8
Type: cpu
Time: 2026-06-26 09:42:16 CST
Duration: 5.68s, Total samples = 21130ms (372.11%)
Showing nodes accounting for 11180ms, 52.91% of 21130ms total
Dropped 299 nodes (cum <= 105.65ms)
Showing top 20 nodes out of 163
      flat  flat%   sum%        cum   cum%
    1560ms  7.38%  7.38%     1680ms  7.95%  github.com/mendixlabs/mxcli/internal/bsoncompare.shouldIgnore (inline)
    1070ms  5.06% 12.45%     1590ms  7.52%  runtime.tryDeferToSpanScan
    1060ms  5.02% 17.46%     4040ms 19.12%  github.com/mendixlabs/mxcli/internal/bsoncompare.normalizeDoc
     660ms  3.12% 20.59%     1750ms  8.28%  runtime.scanObjectsSmall
     630ms  2.98% 23.57%      630ms  2.98%  runtime.nextFreeFast (inline)
     590ms  2.79% 26.36%     1360ms  6.44%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).Load
     580ms  2.74% 29.11%     2750ms 13.01%  runtime.mallocgcSmallScanNoHeader
     550ms  2.60% 31.71%      940ms  4.45%  github.com/mendixlabs/mxcli/internal/bsoncompare.collectIDs
     520ms  2.46% 34.17%     3950ms 18.69%  github.com/mendixlabs/mxcli/internal/bsoncompare.normalizeVal
     520ms  2.46% 36.63%      520ms  2.46%  runtime.memmove
     460ms  2.18% 38.81%      460ms  2.18%  runtime.memclrNoHeapPointers
     420ms  1.99% 40.80%     4300ms 20.35%  runtime.mallocgc
     410ms  1.94% 42.74%    11900ms 56.32%  go.mongodb.org/mongo-driver/v2/bson.decodeTypeOrValueWithInfo
     400ms  1.89% 44.63%    11840ms 56.03%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
     340ms  1.61% 46.24%      390ms  1.85%  runtime.(*mspan).writeHeapBitsSmall
     310ms  1.47% 47.70%      730ms  3.45%  reflect.Value.assignTo
     280ms  1.33% 49.03%      280ms  1.33%  aeshashbody
     280ms  1.33% 50.35%      280ms  1.33%  internal/runtime/syscall/linux.Syscall6
     270ms  1.28% 51.63%    11900ms 56.32%  go.mongodb.org/mongo-driver/v2/bson.(*emptyInterfaceCodec).decodeType
     270ms  1.28% 52.91%      320ms  1.51%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).pop
```

## Top Memory (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: 9fd7dd0ec04791be6794639216676a1f1a946da8
Type: alloc_space
Time: 2026-06-26 09:42:22 CST
Showing nodes accounting for 6246.38MB, 97.99% of 6374.43MB total
Dropped 111 nodes (cum <= 31.87MB)
Showing top 20 nodes out of 56
      flat  flat%   sum%        cum   cum%
 2708.63MB 42.49% 42.49%  4654.93MB 73.03%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
     807MB 12.66% 55.15%      862MB 13.52%  github.com/mendixlabs/mxcli/internal/bsoncompare.normalizeDoc
  538.29MB  8.44% 63.60%   538.29MB  8.44%  os.readFileContents
  476.67MB  7.48% 71.07%   476.67MB  7.48%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).ReadBinary
  362.01MB  5.68% 76.75%   362.01MB  5.68%  reflect.unsafe_New
  302.51MB  4.75% 81.50%   302.51MB  4.75%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readCString
  191.32MB  3.00% 84.50%   191.32MB  3.00%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readString
  176.05MB  2.76% 87.26%  4567.22MB 71.65%  go.mongodb.org/mongo-driver/v2/bson.decodeDefault
  139.50MB  2.19% 89.45%   330.82MB  5.19%  go.mongodb.org/mongo-driver/v2/bson.(*stringCodec).decodeType
  129.50MB  2.03% 91.48%   606.18MB  9.51%  go.mongodb.org/mongo-driver/v2/bson.binaryDecodeType
   71.01MB  1.11% 92.60%    71.01MB  1.11%  reflect.unsafe_NewArray
      70MB  1.10% 93.69%   141.01MB  2.21%  reflect.MakeSlice
      63MB  0.99% 94.68%       63MB  0.99%  reflect.Value.extendSlice
   54.06MB  0.85% 95.53%   179.56MB  2.82%  github.com/mendixlabs/mxcli/internal/bsoncompare.BuildIDMap
   50.50MB  0.79% 96.32%    50.50MB  0.79%  fmt.Sprintf
   37.50MB  0.59% 96.91%   845.99MB 13.27%  github.com/mendixlabs/mxcli/internal/bsoncompare.normalizeVal
      30MB  0.47% 97.38%    78.50MB  1.23%  github.com/mendixlabs/mxcli/internal/bsoncompare.makeLabel
   18.50MB  0.29% 97.67%   125.50MB  1.97%  github.com/mendixlabs/mxcli/internal/bsoncompare.collectIDs
      11MB  0.17% 97.85%   836.98MB 13.13%  github.com/mendixlabs/mxcli/internal/bsoncompare.normalizeArray
    9.32MB  0.15% 97.99%   564.26MB  8.85%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).buildUnitCache
```
