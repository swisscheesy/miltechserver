-- Disposable generation supplement only; not production DDL or target schema proof.
-- Source: docs/OLD/designs/lin_searchbypage_performance_analysis.md:219-227.
-- The approved shops_schema_baseline.sql remains byte-for-byte unchanged.
CREATE MATERIALIZED VIEW public.lookup_lin_niin_mat AS
SELECT nsn.niin, nsn.item_name, army_lin_to_niin.lin
FROM public.nsn
INNER JOIN public.army_lin_to_niin ON nsn.niin = army_lin_to_niin.niin
ORDER BY army_lin_to_niin.lin, nsn.niin;
CREATE UNIQUE INDEX idx_lookup_lin_niin_mat_niin ON public.lookup_lin_niin_mat (niin);
CREATE INDEX idx_lookup_lin_niin_mat_lin ON public.lookup_lin_niin_mat (lin);

-- Synthetic EMPTY compile dependency approved by the remediation controller.
-- No saved production CREATE definition exists in this repository. Only the
-- observed nullable five-field Jet ABI is reproduced, using the pinned baseline
-- tmde_requirements column types and typed NULL for item_name. WITH NO DATA
-- deliberately supplies no runtime TMDE semantics or production join assumption.
-- Exact target catalog/build and TMDE semantics remain operator acceptance gates.
CREATE MATERIALIZED VIEW public.tmde_interval_mat AS
SELECT niin, NULL::text AS item_name, qty_skot, component, "interval"
FROM public.tmde_requirements
WITH NO DATA;
