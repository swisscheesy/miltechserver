# Shops aggregate PMCS history capacity measurements

Measured 2026-10-04 for remediation Task 23 / F35, using the approved design's section 9.3. These measurements concern the complete PMCS history aggregate. They are separate from message synchronization throughput and the Task 28 equipment overview scenario.

## Scenario and method

The guarded `scripts/test-shops-isolated.sh` wrapper created a fresh marker-protected loopback `miltech_test_shops` cluster, restored the approved baseline and generation supplement, applied the available 016–023 candidate migrations, regenerated only its isolated tagged build workspace, and passed the fresh full available-candidate build. No existing database, application pool configuration, schema, index, or generated source was changed for this measurement.

Environment: PostgreSQL 14.18 (Homebrew), Go 1.23.3, darwin/arm64; default `initdb` settings, `work_mem=4MB`, one repository request at a time. The request used a single read-only REPEATABLE READ transaction. Fixtures were inserted locally, then the six involved tables were ANALYZEd. PostgreSQL cache was warm from fixture loading; EXPLAIN ran after the complete repository request, also warm. No planner settings were forced. Plans can use PostgreSQL's default parallel workers; the application still issues component queries sequentially on the same transaction.

Accessible fixture: one Shop and one equipment row, 65,536 inspections (32,768 guide and 32,768 custom), 98,304 faults, and 98,304 comments. Each guide has one fault and two comments; each custom has two faults and one comment. One guide comment is the persisted soft-delete text `Deleted by user` and still counts. Custom records preserve a revision-number-7 snapshot for a retired private checklist with no live authored tree requirement. Foreign controls: a separate Shop/member, one case-different TEXT equipment ID, and one guide plus one custom inspection, each with one fault and comment. Their custom name and private content do not enter the result. There are 65,538 inspections and 98,306 fault/comment rows in total. The capacity regression checks every accessible row's unique ID, date order, source, applicable/absent provenance, author, fault count, and comment count; a subsequent snapshot returns empty after membership removal.

Reproduce from the candidate worktree (with the approved warm Go cache when available):

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh -v -count=1 \
  -run 'TestPmcsHistoryBeyondParameterLimit|TestPmcsHistoryQueryParameterCount' \
  ./tests/shops ./tests/equipment_services
```

The physical pq Connector forwards BeginTx options and counts actual driver arguments; it does not return fake rows. `TestPmcsHistoryQueryParameterCount` additionally checks two and 128 accessible inspections. Before the change the full fixture reached the fault-count query and failed with `got 65536 parameters but PostgreSQL only supports 65535 parameters` from pinned pq v1.10.9. No before-change complete payload exists because that request failed.

## Results

| Measurement | Initial GREEN | Final GREEN |
| --- | ---: | ---: |
| Accessible inspections returned | 65,536 | 65,536 |
| Repository SELECTs / independent author SELECTs | 4 / 0 | 4 / 0 |
| Bound parameters, each of the four SELECTs | 1 | 1 |
| Complete response JSON bytes | 25,886,909 | 25,886,909 |
| Repository wall time | 519.570 ms | 413.187 ms |
| Cumulative Go allocations during repository call | 525,003,880 bytes | 525,032,136 bytes |
| Go HeapAlloc before / after repository call | 3,221,208 / 78,407,488 bytes | 3,183,864 / 75,841,224 bytes |
| Equipment EXPLAIN execution | 0.022 ms | 0.013 ms |
| Inspections with author EXPLAIN execution | 56.932 ms | 43.971 ms |
| Fault counts EXPLAIN execution | 63.852 ms | 40.445 ms |
| Comment counts EXPLAIN execution | 44.421 ms | 42.379 ms |

Memory samples surround the repository call after a runtime GC and before row assertions or JSON encoding. TotalAlloc is cumulative process allocation, including Jet decoding and test-only query instrumentation; HeapAlloc is a point-in-time sample that depends on GC timing, not peak or retained-result memory. JSON size comes from marshaling the complete `EquipmentPmcsHistoryResponse`; encoding time, additional encoding allocations, HTTP envelope/transport, concurrent traffic, and production connection latency were not measured. These two local samples are observations, not a percentile, capacity budget, or production guarantee.

The parameter ceiling is removed without truncation. Full history remains proportional to result size: this sample is approximately 25.9 MB of JSON, roughly 525 MB of cumulative Go allocations, and 73–78 MB of sampled live Go heap. Large concurrent histories may impose material memory/latency pressure; pagination/caps or streaming would change the approved response contract and are outside this repair.

Plans below use the exact four captured repository SQL statements and their one synthetic membership argument. The inspection sort spills to temporary disk; both count hash aggregates use five batches and spill. The fixture selects nearly the entire inspection/fault/comment tables, so sequential scans and hash joins are reasonable here. No speculative index or configuration change was added. A large foreign-data distribution, many-equipment scenario, concurrent histories, peak RSS, or cold-cache production workload has not been measured. Task 28's separate overview/performance release-owner gate remains required.

## Final representative EXPLAIN (ANALYZE, BUFFERS)

### Query 1

```text
Sort  (cost=2.08..2.08 rows=1 width=91) (actual time=0.006..0.007 rows=1 loops=1)
  Sort Key: shop_vehicle.save_time DESC, shop_vehicle.id DESC
  Sort Method: quicksort  Memory: 25kB
  Buffers: shared hit=2
  ->  Nested Loop  (cost=0.00..2.07 rows=1 width=91) (actual time=0.004..0.004 rows=1 loops=1)
        Join Filter: (shop_vehicle.shop_id = shop_members.shop_id)
        Rows Removed by Join Filter: 1
        Buffers: shared hit=2
        ->  Seq Scan on shop_members  (cost=0.00..1.02 rows=1 width=37) (actual time=0.002..0.002 rows=1 loops=1)
              Filter: (user_id = 'pmcs-capacity-member'::text)
              Rows Removed by Filter: 1
              Buffers: shared hit=1
        ->  Seq Scan on shop_vehicle  (cost=0.00..1.02 rows=2 width=91) (actual time=0.001..0.001 rows=2 loops=1)
              Buffers: shared hit=1
Planning:
  Buffers: shared hit=189
Planning Time: 0.409 ms
Execution Time: 0.013 ms

```

### Query 2

```text
Gather Merge  (cost=7201.23..9417.97 rows=19276 width=250) (actual time=32.530..42.003 rows=65536 loops=1)
  Workers Planned: 1
  Workers Launched: 1
  Buffers: shared hit=1745, temp read=1558 written=1562
  ->  Sort  (cost=6201.22..6249.41 rows=19276 width=250) (actual time=30.112..31.797 rows=32768 loops=2)
        Sort Key: user_pmcs_inspections.equipment_id, user_pmcs_inspections.performed_date DESC
        Sort Method: external merge  Disk: 6760kB
        Buffers: shared hit=1745, temp read=1558 written=1562
        Worker 0:  Sort Method: external merge  Disk: 5704kB
        ->  Hash Left Join  (cost=3.13..2522.80 rows=19276 width=250) (actual time=0.047..8.066 rows=32768 loops=2)
              Hash Cond: (user_pmcs_inspections.performed_by = users.uid)
              Buffers: shared hit=1698
              ->  Hash Join  (cost=2.08..2363.93 rows=19276 width=238) (actual time=0.037..5.392 rows=32768 loops=2)
                    Hash Cond: (user_pmcs_inspections.equipment_id = shop_vehicle.id)
                    Buffers: shared hit=1696
                    ->  Parallel Seq Scan on user_pmcs_inspections  (cost=0.00..2024.52 rows=38552 width=238) (actual time=0.002..1.594 rows=32769 loops=2)
                          Buffers: shared hit=1639
                    ->  Hash  (cost=2.07..2.07 rows=1 width=37) (actual time=0.018..0.018 rows=1 loops=2)
                          Buckets: 1024  Batches: 1  Memory Usage: 9kB
                          Buffers: shared hit=4
                          ->  Nested Loop  (cost=0.00..2.07 rows=1 width=37) (actual time=0.015..0.016 rows=1 loops=2)
                                Join Filter: (shop_vehicle.shop_id = shop_members.shop_id)
                                Rows Removed by Join Filter: 1
                                Buffers: shared hit=4
                                ->  Seq Scan on shop_members  (cost=0.00..1.02 rows=1 width=37) (actual time=0.008..0.008 rows=1 loops=2)
                                      Filter: (user_id = 'pmcs-capacity-member'::text)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=2
                                ->  Seq Scan on shop_vehicle  (cost=0.00..1.02 rows=2 width=74) (actual time=0.007..0.007 rows=2 loops=2)
                                      Buffers: shared hit=2
              ->  Hash  (cost=1.02..1.02 rows=2 width=33) (actual time=0.008..0.008 rows=2 loops=2)
                    Buckets: 1024  Batches: 1  Memory Usage: 9kB
                    Buffers: shared hit=2
                    ->  Seq Scan on users  (cost=0.00..1.02 rows=2 width=33) (actual time=0.007..0.007 rows=2 loops=2)
                          Buffers: shared hit=2
Planning:
  Buffers: shared hit=140
Planning Time: 0.239 ms
Execution Time: 43.971 ms

```

### Query 3

```text
HashAggregate  (cost=9582.63..10554.17 rows=49153 width=24) (actual time=31.496..38.944 rows=65536 loops=1)
  Group Key: user_pmcs_faults.pmcs_id
  Batches: 5  Memory Usage: 4145kB  Disk Usage: 3360kB
  Buffers: shared hit=2952, temp read=241 written=595
  ->  Hash Join  (cost=3279.53..6433.77 rows=49153 width=16) (actual time=10.416..21.643 rows=98304 loops=1)
        Hash Cond: (user_pmcs_faults.pmcs_id = user_pmcs_inspections.id)
        Buffers: shared hit=2952
        ->  Seq Scan on user_pmcs_faults  (cost=0.00..2294.06 rows=98306 width=16) (actual time=0.002..2.874 rows=98306 loops=1)
              Buffers: shared hit=1311
        ->  Hash  (cost=2869.92..2869.92 rows=32769 width=16) (actual time=10.411..10.411 rows=65536 loops=1)
              Buckets: 65536  Batches: 1  Memory Usage: 3584kB
              Buffers: shared hit=1641
              ->  Hash Join  (cost=2.08..2869.92 rows=32769 width=16) (actual time=0.006..7.515 rows=65536 loops=1)
                    Hash Cond: (user_pmcs_inspections.equipment_id = shop_vehicle.id)
                    Buffers: shared hit=1641
                    ->  Seq Scan on user_pmcs_inspections  (cost=0.00..2294.38 rows=65538 width=53) (actual time=0.001..2.315 rows=65538 loops=1)
                          Buffers: shared hit=1639
                    ->  Hash  (cost=2.07..2.07 rows=1 width=37) (actual time=0.004..0.004 rows=1 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 9kB
                          Buffers: shared hit=2
                          ->  Nested Loop  (cost=0.00..2.07 rows=1 width=37) (actual time=0.002..0.003 rows=1 loops=1)
                                Join Filter: (shop_vehicle.shop_id = shop_members.shop_id)
                                Rows Removed by Join Filter: 1
                                Buffers: shared hit=2
                                ->  Seq Scan on shop_members  (cost=0.00..1.02 rows=1 width=37) (actual time=0.001..0.001 rows=1 loops=1)
                                      Filter: (user_id = 'pmcs-capacity-member'::text)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=1
                                ->  Seq Scan on shop_vehicle  (cost=0.00..1.02 rows=2 width=74) (actual time=0.001..0.001 rows=2 loops=1)
                                      Buffers: shared hit=1
Planning:
  Buffers: shared hit=100
Planning Time: 0.265 ms
Execution Time: 40.445 ms

```

### Query 4

```text
HashAggregate  (cost=9582.63..10554.17 rows=49153 width=24) (actual time=31.806..39.033 rows=65536 loops=1)
  Group Key: user_pmcs_inspection_comments.pmcs_id
  Batches: 5  Memory Usage: 4145kB  Disk Usage: 3360kB
  Buffers: shared hit=2952, temp read=240 written=592
  ->  Hash Join  (cost=3279.53..6433.77 rows=49153 width=16) (actual time=10.417..22.086 rows=98304 loops=1)
        Hash Cond: (user_pmcs_inspection_comments.pmcs_id = user_pmcs_inspections.id)
        Buffers: shared hit=2952
        ->  Seq Scan on user_pmcs_inspection_comments  (cost=0.00..2294.06 rows=98306 width=16) (actual time=0.002..2.884 rows=98306 loops=1)
              Buffers: shared hit=1311
        ->  Hash  (cost=2869.92..2869.92 rows=32769 width=16) (actual time=10.411..10.412 rows=65536 loops=1)
              Buckets: 65536  Batches: 1  Memory Usage: 3584kB
              Buffers: shared hit=1641
              ->  Hash Join  (cost=2.08..2869.92 rows=32769 width=16) (actual time=0.005..7.466 rows=65536 loops=1)
                    Hash Cond: (user_pmcs_inspections.equipment_id = shop_vehicle.id)
                    Buffers: shared hit=1641
                    ->  Seq Scan on user_pmcs_inspections  (cost=0.00..2294.38 rows=65538 width=53) (actual time=0.001..2.316 rows=65538 loops=1)
                          Buffers: shared hit=1639
                    ->  Hash  (cost=2.07..2.07 rows=1 width=37) (actual time=0.003..0.004 rows=1 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 9kB
                          Buffers: shared hit=2
                          ->  Nested Loop  (cost=0.00..2.07 rows=1 width=37) (actual time=0.002..0.003 rows=1 loops=1)
                                Join Filter: (shop_vehicle.shop_id = shop_members.shop_id)
                                Rows Removed by Join Filter: 1
                                Buffers: shared hit=2
                                ->  Seq Scan on shop_members  (cost=0.00..1.02 rows=1 width=37) (actual time=0.001..0.001 rows=1 loops=1)
                                      Filter: (user_id = 'pmcs-capacity-member'::text)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=1
                                ->  Seq Scan on shop_vehicle  (cost=0.00..1.02 rows=2 width=74) (actual time=0.000..0.001 rows=2 loops=1)
                                      Buffers: shared hit=1
Planning:
  Buffers: shared hit=99
Planning Time: 0.254 ms
Execution Time: 42.379 ms
```
