# Performance Profile: ./internal/bsoncompare/

| Metric | Value |
|--------|-------|
| Package | `./internal/bsoncompare/` |
| Timestamp | 20260626_095133 |
| Wall time | 10s |
| Tests | 32 |
| Slowest test | 7.83s |
| Coverage | 75.9% |
| Exit code | 0 |

## Slowest Tests

```
7.83 TestCompare_ContentHashSkip (7.83s)
7.78 TestBuildIDMap_CorpusB (7.78s)
7.73 TestReadAllUnits_CorpusB (7.73s)
7.59 TestCompare_NoChange (7.59s)
7.58 TestReadAllUnits_ContentHashPopulated (7.58s)
7.50 TestReadAllUnits_ContentHashStable (7.50s)
7.47 TestAssertEqual_SelfComparePasses (7.47s)
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

- CPU: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__cpu_20260626_095133.pprof`
- Memory: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__mem_20260626_095133.pprof`
- Coverage: `/mnt/data_sdb/mxcli/profile/internal_bsoncompare__cover_20260626_095133.out`

## Top CPU (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: dca69df0a48cad9aff58ec62f73b5ab114618506
Type: cpu
Time: 2026-06-26 09:51:34 CST
Duration: 7.83s, Total samples = 44440ms (567.54%)
Showing nodes accounting for 21130ms, 47.55% of 44440ms total
Dropped 343 nodes (cum <= 222.20ms)
Showing top 20 nodes out of 155
      flat  flat%   sum%        cum   cum%
    1890ms  4.25%  4.25%     4770ms 10.73%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).Load
    1680ms  3.78%  8.03%     1680ms  3.78%  runtime.memmove
    1600ms  3.60% 11.63%     2390ms  5.38%  runtime.tryDeferToSpanScan
    1560ms  3.51% 15.14%     1560ms  3.51%  runtime.nextFreeFast (inline)
    1470ms  3.31% 18.45%    11480ms 25.83%  runtime.mallocgc
    1390ms  3.13% 21.58%     7130ms 16.04%  runtime.mallocgcSmallScanNoHeader
    1380ms  3.11% 24.68%     1500ms  3.38%  runtime.(*mspan).writeHeapBitsSmall
    1030ms  2.32% 27.00%     2580ms  5.81%  runtime.scanObjectsSmall
     990ms  2.23% 29.23%    36670ms 82.52%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
     990ms  2.23% 31.46%      990ms  2.23%  runtime.memclrNoHeapPointers
     890ms  2.00% 33.46%      890ms  2.00%  hash/fnv.(*sum64a).Write
     820ms  1.85% 35.31%     2450ms  5.51%  reflect.Value.assignTo
     810ms  1.82% 37.13%    36700ms 82.58%  go.mongodb.org/mongo-driver/v2/bson.(*emptyInterfaceCodec).decodeType
     770ms  1.73% 38.86%      770ms  1.73%  internal/runtime/syscall/linux.Syscall6
     750ms  1.69% 40.55%    36660ms 82.49%  go.mongodb.org/mongo-driver/v2/bson.decodeTypeOrValueWithInfo
     660ms  1.49% 42.03%      840ms  1.89%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).pop
     630ms  1.42% 43.45%      970ms  2.18%  go.mongodb.org/mongo-driver/v2/bson.readBytes
     630ms  1.42% 44.87%      630ms  1.42%  indexbytebody
     610ms  1.37% 46.24%     1510ms  3.40%  runtime.mallocgcTiny
     580ms  1.31% 47.55%     3070ms  6.91%  go.mongodb.org/mongo-driver/v2/bson.(*Registry).LookupTypeMapEntry
```

## Top Memory (pprof -top -nodecount=20)

```
File: bsoncompare.test
Build ID: dca69df0a48cad9aff58ec62f73b5ab114618506
Type: alloc_space
Time: 2026-06-26 09:51:42 CST
Showing nodes accounting for 9188.25MB, 98.52% of 9326.67MB total
Dropped 105 nodes (cum <= 46.63MB)
Showing top 20 nodes out of 51
      flat  flat%   sum%        cum   cum%
 4811.47MB 51.59% 51.59%  8163.47MB 87.53%  go.mongodb.org/mongo-driver/v2/bson.dDecodeValue
  948.45MB 10.17% 61.76%   948.45MB 10.17%  os.readFileContents
  820.15MB  8.79% 70.55%   820.15MB  8.79%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).ReadBinary
  629.01MB  6.74% 77.30%   629.01MB  6.74%  reflect.unsafe_New
  547.01MB  5.87% 83.16%   547.01MB  5.87%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readCString
  301.81MB  3.24% 86.40%   301.81MB  3.24%  go.mongodb.org/mongo-driver/v2/bson.(*valueReader).readString
  275.58MB  2.95% 89.35%  8025.91MB 86.05%  go.mongodb.org/mongo-driver/v2/bson.decodeDefault
     235MB  2.52% 91.87%   536.82MB  5.76%  go.mongodb.org/mongo-driver/v2/bson.(*stringCodec).decodeType
  212.01MB  2.27% 94.14%  1032.16MB 11.07%  go.mongodb.org/mongo-driver/v2/bson.binaryDecodeType
  131.02MB  1.40% 95.55%   131.02MB  1.40%  reflect.unsafe_NewArray
     125MB  1.34% 96.89%   256.02MB  2.75%  reflect.MakeSlice
  111.50MB  1.20% 98.08%   111.50MB  1.20%  reflect.Value.extendSlice
   14.86MB  0.16% 98.24%   998.65MB 10.71%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).buildUnitCache
   10.12MB  0.11% 98.35%  1008.77MB 10.82%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).listUnitsByTypeV2
    8.83MB 0.095% 98.45%   973.29MB 10.44%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).readMprContents
    3.42MB 0.037% 98.48%  7197.32MB 77.17%  github.com/mendixlabs/mxcli/internal/bsoncompare.ReadAllUnits
       3MB 0.032% 98.52%  4092.26MB 43.88%  github.com/mendixlabs/mxcli/modelsdk/mpr.(*Reader).ListRawUnits
         0     0% 98.52%  1010.55MB 10.84%  github.com/mendixlabs/mxcli/internal/bsoncompare.AssertEqual
         0     0% 98.52%  2046.31MB 21.94%  github.com/mendixlabs/mxcli/internal/bsoncompare.Compare
         0     0% 98.52%   959.52MB 10.29%  github.com/mendixlabs/mxcli/internal/bsoncompare_test.TestAssertEqual_SelfComparePasses
```
