--
-- PostgreSQL database dump
--

-- Dumped from database version 14.18 (Debian 14.18-1.pgdg120+1)
-- Dumped by pg_dump version 14.18 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: air_force_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.air_force_management (
    niin character varying(255) NOT NULL,
    fund character varying(255),
    budget character varying(255),
    mmac character varying(255),
    pvc character varying(255),
    errc character varying(255)
);


--
-- Name: amdf_billing; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_billing (
    niin character varying(255) NOT NULL,
    serviceable_credit_value character varying(255),
    unserviceable_credit_value character varying(255),
    exchange_price character varying(255),
    serviceable_ep_return character varying(255),
    delta_bill character varying(255)
);


--
-- Name: amdf_credit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_credit (
    niin character varying(255) NOT NULL,
    aril character varying(255),
    aril_ric character varying(255),
    demil_code character varying(255),
    adpe_code character varying(255),
    pmic character varying(255),
    mr character varying(255),
    recov_code character varying(255),
    esd character varying(255),
    hmic character varying(255),
    critl_code character varying(255)
);


--
-- Name: amdf_freight; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_freight (
    niin character varying(255) NOT NULL,
    item_description character varying(255),
    nmfc character varying(255),
    nmfc_sub character varying(255),
    ltl character varying(255),
    nmf_desc character varying(255),
    stc character varying(255),
    lcl character varying(255),
    rvc character varying(255),
    ufc character varying(255),
    adc character varying(255),
    acc character varying(255),
    wcc character varying(255),
    shc character varying(255),
    tcc character varying(255)
);


--
-- Name: amdf_i_and_s; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_i_and_s (
    id integer NOT NULL,
    niin character varying(255),
    oou character varying(255),
    jtc character varying(255),
    related_fsc character varying(255),
    related_niin character varying(255)
);


--
-- Name: amdf_i_and_s_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.amdf_i_and_s_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: amdf_i_and_s_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.amdf_i_and_s_id_seq OWNED BY public.amdf_i_and_s.id;


--
-- Name: amdf_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_management (
    niin character varying(255) NOT NULL,
    scmc character varying(255),
    aec character varying(255),
    matcat character varying(255),
    lin character varying(255),
    lcc character varying(255),
    ricc character varying(255),
    arc character varying(255),
    src character varying(255),
    scic character varying(255),
    ciic character varying(255),
    icc character varying(255),
    slc character varying(255)
);


--
-- Name: amdf_matcat; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_matcat (
    niin character varying(255) NOT NULL,
    matcat_1 character varying(255),
    matcat_2 character varying(255),
    matcat_3 character varying(255),
    matcat_4_5 character varying(255)
);


--
-- Name: amdf_phrase; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.amdf_phrase (
    id integer NOT NULL,
    niin character varying(255),
    phrase_code character varying(255),
    phrase_statement character varying(255),
    ui_rel character varying(255),
    um_rel character varying(255),
    qty_per_assy character varying(255)
);


--
-- Name: amdf_phrase_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.amdf_phrase_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: amdf_phrase_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.amdf_phrase_id_seq OWNED BY public.amdf_phrase.id;


--
-- Name: analytics_event_counters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.analytics_event_counters (
    id character varying(36) NOT NULL,
    event_type text NOT NULL,
    entity_key text NOT NULL,
    entity_label text,
    count bigint DEFAULT 1 NOT NULL,
    last_seen_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT chk_count_non_negative CHECK ((count >= 0))
);


--
-- Name: army_freight; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_freight (
    niin character varying(255) NOT NULL,
    un_number character varying(255),
    msds_indicator character varying(255)
);


--
-- Name: army_lin_to_niin; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_lin_to_niin (
    lin character varying(255),
    niin character varying(255) NOT NULL
);


--
-- Name: army_line_item_number; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_line_item_number (
    lin character varying(255) NOT NULL,
    type_action character varying(255),
    cic character varying(255),
    cmc character varying(255),
    ric character varying(255),
    pub_date character varying(255),
    aps character varying(255),
    lin_delete_statement character varying(255),
    new_lin character varying(255),
    and_or character varying(255),
    repl_ratio character varying(255),
    type_class character varying(255),
    add_chg_del_date character varying(255),
    nsn_delete_statement character varying(255),
    assigned_niin character varying(255)
);


--
-- Name: army_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_management (
    id integer NOT NULL,
    niin character varying(255),
    matcat_1 character varying(255),
    matcat_2 character varying(255),
    matcat_3 character varying(255),
    matcat_4_5 character varying(255),
    arc character varying(255)
);


--
-- Name: army_management_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.army_management_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: army_management_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.army_management_id_seq OWNED BY public.army_management.id;


--
-- Name: army_master_data_file; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_master_data_file (
    niin character varying(255) NOT NULL,
    fsc character varying(255),
    nomenclature character varying(255),
    act character varying(255),
    addl character varying(255),
    sos character varying(255),
    aac character varying(255),
    psc character varying(255),
    army_unit_price character varying(255),
    ui character varying(255),
    fc character varying(255),
    um character varying(255),
    meas_qty character varying(255),
    eic character varying(255),
    ec character varying(255)
);


--
-- Name: army_pack_supplemental_instruct; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_pack_supplemental_instruct (
    niin character varying(255) NOT NULL,
    supplemental_instructions character varying(255)
);


--
-- Name: army_packaging_1; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_packaging_1 (
    niin character varying(255) NOT NULL,
    mop character varying(255),
    clng_drying character varying(255),
    pres_mat character varying(255),
    wrap_mat character varying(255),
    cush_dun character varying(255),
    thk character varying(255),
    unit_cont character varying(255),
    inter_cont character varying(255),
    opi character varying(255),
    spc_mkg character varying(255),
    ucl character varying(255),
    lvl_a character varying(255),
    lvl_b character varying(255),
    lvl_c character varying(255)
);


--
-- Name: army_packaging_2; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_packaging_2 (
    niin character varying(255) NOT NULL,
    pkg_cat character varying(255),
    unpkg_item_weight character varying(255),
    unpkg_item_dim character varying(255),
    drwg_pn character varying(255),
    cage_code character varying(255)
);


--
-- Name: army_packaging_and_freight; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_packaging_and_freight (
    niin character varying(255) NOT NULL,
    lop character varying(255),
    pkg_ref character varying(255),
    upq character varying(255),
    icq character varying(255),
    tos character varying(255),
    haz character varying(255),
    unit_pack_size character varying(255),
    unit_pack_weight character varying(255),
    unit_pack_cube character varying(255),
    pkg_ind character varying(255),
    pk_lvl_ref_ind character varying(255)
);


--
-- Name: army_packaging_special_instruct; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_packaging_special_instruct (
    niin character varying(255) NOT NULL,
    cont_nsn character varying(255),
    spi_no character varying(255),
    spi_rev character varying(255),
    spi_date character varying(255),
    pkg_design_acty character varying(255)
);


--
-- Name: army_related_nsn; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_related_nsn (
    niin character varying(255) NOT NULL,
    lin character varying(255),
    nomenclature character varying(255),
    related_niin character varying(255),
    tfc character varying(255),
    army_type_designator character varying(255),
    type_class character varying(255),
    mscr character varying(255),
    reference_data character varying(255)
);


--
-- Name: army_sarsscat; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_sarsscat (
    niin character varying(255) NOT NULL,
    in_code character varying(255),
    muc character varying(255),
    wrty character varying(255),
    ui_old character varying(255),
    ui_conv_factor character varying(255),
    aimi character varying(255),
    lop character varying(255),
    sp_strg character varying(255),
    temp character varying(255),
    slc character varying(255),
    related_niin character varying(255),
    aril_ric_1 character varying(255),
    aril_ric_2 character varying(255),
    aril_ric_3 character varying(255),
    aril_ric_4 character varying(255),
    aril_ric_5 character varying(255)
);


--
-- Name: army_substitute_lin; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.army_substitute_lin (
    lin character varying(255) NOT NULL,
    substitute_lin character varying(255) NOT NULL
);


--
-- Name: cage_address; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cage_address (
    cage_code character varying(255) NOT NULL,
    company_name text,
    company_name_2 text,
    company_name_3 text,
    company_name_4 text,
    company_name_5 text,
    street_address_1 text,
    street_address_2 text,
    po_box text,
    city text,
    state text,
    zip text,
    country text,
    date_est text,
    last_update text,
    former_name_1 text,
    former_name_2 text,
    former_name_3 text,
    former_name_4 text,
    frn_dom text
);


--
-- Name: cage_status_and_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cage_status_and_type (
    cage_code character varying(255) NOT NULL,
    status character varying(255),
    type character varying(255),
    cao character varying(255),
    adp character varying(255),
    phone character varying(255),
    fax character varying(255),
    rplm_code character varying(255),
    assoc_code character varying(255),
    affil_code character varying(255),
    bus_size character varying(255),
    primary_business character varying(255),
    type_of_business character varying(255),
    woman_owned character varying(255),
    cngrsl_dstrct character varying(255),
    designator character varying(255)
);


--
-- Name: coast_guard_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.coast_guard_management (
    niin character varying(255) NOT NULL,
    iac character varying(255),
    snc character varying(255),
    smcc character varying(255)
);


--
-- Name: colloquial_name; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.colloquial_name (
    inc character varying(255) NOT NULL,
    related_inc character varying(255) NOT NULL,
    colloquial_name character varying(255) NOT NULL
);


--
-- Name: component_end_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.component_end_item (
    niin character varying(255) NOT NULL,
    wpn_sys_id character varying(255) NOT NULL,
    wpn_sys_svc character varying(255) NOT NULL,
    wpn_sys_ind character varying(255),
    wpn_sys_esntl character varying(255),
    weapon_system character varying(255)
);


--
-- Name: disposition; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.disposition (
    niin character varying(255) NOT NULL,
    demil_code character varying(255),
    demil_intg character varying(255),
    fsc_flag_type character varying(255),
    niin_flag_type character varying(255),
    other_flag_type character varying(255),
    do_not_sell character varying(255),
    safe_to_sell character varying(255)
);


--
-- Name: TABLE disposition; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.disposition IS '"DRMS Disposal"';


--
-- Name: docs_equipment_details; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.docs_equipment_details (
    model text,
    lin text,
    mode text,
    description text,
    length text,
    width text,
    height text,
    weight text,
    kw integer,
    hz text,
    family text,
    id bigint NOT NULL
);


--
-- Name: docs_equipment_details_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.docs_equipment_details_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: docs_equipment_details_id_seq1; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.docs_equipment_details ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.docs_equipment_details_id_seq1
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: dss_weight_and_cube; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dss_weight_and_cube (
    niin character varying(255) NOT NULL,
    dss_weight character varying(255),
    dss_cube character varying(255)
);


--
-- Name: eic; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.eic (
    inc character varying(255),
    fsc character varying(255),
    niin character varying(255) NOT NULL,
    eic character varying(255),
    uoeic character varying(255) NOT NULL,
    lin character varying(255),
    nomen character varying(255),
    model character varying(255),
    eicc character varying(255),
    ecc character varying(255),
    cmdtycd character varying(255),
    reported character varying(255),
    dahr character varying(255),
    publvl1 character varying(255),
    pubno1 character varying(255),
    pubdate1 character varying(255),
    pubchg1 character varying(255),
    pubcgdt1 character varying(255),
    publcl2 character varying(255),
    pubno2 character varying(255),
    pubdate2 character varying(255),
    pubchg2 character varying(255),
    pubcgdt2 character varying(255),
    publvl3 character varying(255),
    pubno3 character varying(255),
    pubdate3 character varying(255),
    pubchg3 character varying(255),
    pubcgdt3 character varying(255),
    publvl4 character varying(255),
    pubno4 character varying(255),
    pubdate4 character varying(255),
    pubchg4 character varying(255),
    pubcgdt4 character varying(255),
    publvl5 character varying(255),
    pubno5 character varying(255),
    pubdate5 character varying(255),
    pubchg5 character varying(255),
    pubcgdt5 character varying(255),
    publvl6 character varying(255),
    pubno6 character varying(255),
    pubdate6 character varying(255),
    pubchg6 character varying(255),
    pubcgdt6 character varying(255),
    publvl7 character varying(255),
    pubno7 character varying(255),
    pubdate7 character varying(255),
    pubchg7 character varying(255),
    pubcgdt7 character varying(255),
    pubremks character varying(255),
    eqpmcsa character varying(255),
    eqpmcsb character varying(255),
    eqpmcsc character varying(255),
    eqpmcsd character varying(255),
    eqpmcse character varying(255),
    eqpmcsf character varying(255),
    eqpmcsg character varying(255),
    eqpmcsh character varying(255),
    eqpmcsi character varying(255),
    eqpmcsj character varying(255),
    eqpmcsk character varying(255),
    eqpmcsl character varying(255),
    wpnrec character varying(255),
    sernotrk character varying(255),
    orf character varying(255),
    aoap character varying(255),
    gainloss character varying(255),
    usage character varying(255),
    urm1 character varying(255),
    urm2 character varying(255),
    uom1 character varying(255),
    uom2 character varying(255),
    uom3 character varying(255),
    mau1 character varying(255),
    uom4 character varying(255),
    mau2 character varying(255),
    warranty character varying(255),
    rbm character varying(255),
    mrc character varying(255) NOT NULL,
    sos character varying(255),
    erc character varying(255),
    eslvl character varying(255),
    oslin character varying(255),
    lcc character varying(255),
    nounabb character varying(255),
    curfmc character varying(255),
    prevfmc character varying(255),
    bstat1 character varying(255),
    bstat2 character varying(255),
    matcat character varying(255),
    itemmgr character varying(255),
    eos character varying(255),
    sorts character varying(255),
    status character varying(255),
    lst_updt character varying(255)
);


--
-- Name: equipment_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.equipment_services (
    id character varying(36) NOT NULL,
    shop_id character varying(36) NOT NULL,
    equipment_id character varying(36) NOT NULL,
    list_id character varying(36) NOT NULL,
    description text NOT NULL,
    service_type text NOT NULL,
    created_by character varying(255) NOT NULL,
    is_completed boolean DEFAULT false NOT NULL,
    created_at timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp(6) with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    service_date timestamp(6) with time zone,
    service_hours integer,
    completion_date timestamp(6) with time zone,
    CONSTRAINT chk_description_length CHECK (((length(description) >= 1) AND (length(description) <= 500))),
    CONSTRAINT chk_service_hours_non_negative CHECK (((service_hours IS NULL) OR (service_hours >= 0)))
);


--
-- Name: faa_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.faa_management (
    id integer NOT NULL,
    niin character varying(255),
    retail_price character varying(255),
    dod_price character varying(255),
    repair_price character varying(255),
    moe character varying(255)
);


--
-- Name: faa_management_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.faa_management_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: faa_management_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.faa_management_id_seq OWNED BY public.faa_management.id;


--
-- Name: flis_cancelled_niin; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_cancelled_niin (
    id integer NOT NULL,
    niin character varying(255),
    cancelled_niin_fsc character varying(255),
    cancelled_niin character varying(255),
    niin_stat_cd character varying(255),
    eff_date character varying(255),
    demil character varying(255)
);


--
-- Name: flis_cancelled_niin_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_cancelled_niin_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_cancelled_niin_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_cancelled_niin_id_seq OWNED BY public.flis_cancelled_niin.id;


--
-- Name: flis_freight; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_freight (
    niin character varying(255) NOT NULL,
    acty_cd character varying(255),
    integ character varying(255),
    nmfc character varying(255),
    nmfc_sub character varying(255),
    ufc character varying(255),
    rvc character varying(255),
    hmc character varying(255),
    ltl character varying(255),
    lcl character varying(255),
    wcc character varying(255),
    tcc character varying(255),
    shc character varying(255),
    adc character varying(255),
    acc character varying(255),
    ash character varying(255),
    nmf_desc character varying(255)
);


--
-- Name: flis_identification; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_identification (
    niin character varying(255) NOT NULL,
    type_ii character varying(255),
    inc character varying(255),
    hcc character varying(255),
    isc character varying(255),
    standard_niin character varying(255)
);


--
-- Name: flis_item_characteristics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_item_characteristics (
    id integer NOT NULL,
    niin text,
    mrc text,
    requirements_statement text,
    clear_text_reply text
);


--
-- Name: flis_item_characteristics_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_item_characteristics_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_item_characteristics_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_item_characteristics_id_seq OWNED BY public.flis_item_characteristics.id;


--
-- Name: flis_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_management (
    id integer NOT NULL,
    niin character varying(255),
    effective_date character varying(255),
    moe character varying(255),
    sos character varying(255),
    sosm character varying(255),
    aac character varying(255),
    qup character varying(255),
    ui character varying(255),
    ui_conv_fac character varying(255),
    unit_price character varying(255),
    slc character varying(255),
    ciic character varying(255),
    rec_rep_code character varying(255),
    mgmt_ctl character varying(255),
    rep_net_pr character varying(255),
    usc character varying(255)
);


--
-- Name: flis_management_id; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_management_id (
    niin character varying(255) NOT NULL,
    fiig character varying(255),
    pmic character varying(255),
    adpe_code character varying(255),
    critl_code character varying(255),
    rpd_mrc character varying(255),
    demil_code character varying(255),
    demil_intg character varying(255),
    niin_asgmt character varying(255),
    est_act character varying(255),
    est_act_date character varying(255),
    esd character varying(255),
    hmic character varying(255),
    enac character varying(255),
    schedule_b character varying(255),
    inc character varying(255),
    pinc character varying(255),
    min_rlse_qty character varying(255),
    sla character varying(255),
    ui_conv_factor character varying(255),
    fedmall character varying(255),
    iuid_indicator character varying(255),
    lst_kwn_sos character varying(255),
    nato_fmsn character varying(255)
);


--
-- Name: flis_management_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_management_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_management_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_management_id_seq OWNED BY public.flis_management.id;


--
-- Name: flis_packaging_1; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_packaging_1 (
    id integer NOT NULL,
    niin character varying(255),
    pica_sica character varying(255),
    ui character varying(255),
    tos character varying(255),
    icq character varying(255),
    mop character varying(255),
    clng_drying character varying(255),
    pres_mat character varying(255),
    wrap_mat character varying(255),
    pkg_cat character varying(255),
    pkg_design_acty character varying(255),
    cush_dun character varying(255),
    thk character varying(255),
    unit_cont character varying(255),
    pkg_data_source character varying(255),
    inter_cont character varying(255),
    ucl character varying(255)
);


--
-- Name: flis_packaging_1_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_packaging_1_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_packaging_1_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_packaging_1_id_seq OWNED BY public.flis_packaging_1.id;


--
-- Name: flis_packaging_2; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_packaging_2 (
    id integer NOT NULL,
    niin character varying(255),
    pica_sica character varying(255),
    spc_mkg character varying(255),
    lvl_a character varying(255),
    lvl_b character varying(255),
    lvl_c character varying(255),
    unit_pack_weight character varying(255),
    unit_pack_size character varying(255),
    unit_pack_cube character varying(255),
    cont_nsn character varying(255),
    opi character varying(255),
    unpkg_item_dim character varying(255),
    unpkg_item_weight character varying(255),
    spi_date character varying(255),
    spi_no character varying(255),
    spi_rev character varying(255),
    supplemental_instructions character varying(255)
);


--
-- Name: flis_packaging_2_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_packaging_2_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_packaging_2_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_packaging_2_id_seq OWNED BY public.flis_packaging_2.id;


--
-- Name: flis_phrase; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_phrase (
    id integer NOT NULL,
    niin character varying(255),
    moe character varying(255),
    usc character varying(255),
    phrase_code character varying(255),
    phrase_statement character varying(255),
    oou character varying(255),
    jtc character varying(255),
    qpa character varying(255),
    um character varying(255),
    tech_doc_nbr character varying(255),
    qntv_exprsn character varying(255)
);


--
-- Name: flis_phrase_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_phrase_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_phrase_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_phrase_id_seq OWNED BY public.flis_phrase.id;


--
-- Name: flis_reference; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_reference (
    niin character varying(255) NOT NULL,
    part_number character varying(255) NOT NULL,
    cage_code character varying(255) NOT NULL,
    status character varying(255),
    rncc character varying(255),
    rnvc character varying(255),
    dac character varying(255),
    rnaac character varying(255),
    rnfc character varying(255),
    rnsc character varying(255),
    rnjc character varying(255),
    msds character varying(255),
    sadc character varying(255),
    medals character varying(255)
);


--
-- Name: flis_standardization; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flis_standardization (
    id integer NOT NULL,
    niin character varying(255),
    related_nsn character varying(255),
    isc character varying(255),
    orig_stdzn_dec character varying(255),
    dt_stdzn_dec character varying(255),
    niin_stat_cd character varying(255)
);


--
-- Name: flis_standardization_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.flis_standardization_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: flis_standardization_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.flis_standardization_id_seq OWNED BY public.flis_standardization.id;


--
-- Name: help; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.help (
    code text NOT NULL,
    literal text,
    description text NOT NULL,
    regs text
);


--
-- Name: item_comment_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_comment_flags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    comment_id uuid NOT NULL,
    flagger_id text NOT NULL,
    created_at timestamp(6) without time zone DEFAULT now() NOT NULL
);


--
-- Name: item_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    comment_niin text NOT NULL,
    author_id text NOT NULL,
    text character varying(255) NOT NULL,
    parent_id uuid,
    created_at timestamp(6) without time zone DEFAULT now() NOT NULL,
    updated_at timestamp(6) without time zone
);


--
-- Name: nsn; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nsn (
    service character varying(255) NOT NULL,
    category character varying(255) NOT NULL,
    fsc character varying(255) NOT NULL,
    niin character varying(10) NOT NULL,
    cancelled_niin text,
    item_name character varying(255),
    inc character varying(255)
);


--
-- Name: lookup_lin_niin; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.lookup_lin_niin AS
 SELECT nsn.niin,
    nsn.item_name,
    army_lin_to_niin.lin
   FROM public.nsn,
    public.army_lin_to_niin
  WHERE ((nsn.niin)::text = (army_lin_to_niin.niin)::text);


--
-- Name: lookup_uoc; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lookup_uoc (
    uoc character varying(5) NOT NULL,
    model character varying(255) NOT NULL
);


--
-- Name: TABLE lookup_uoc; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lookup_uoc IS 'Table provides a quick access to equipment UOCs by Equipment Model';


--
-- Name: COLUMN lookup_uoc.uoc; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lookup_uoc.uoc IS 'Usable On Code for the equipment';


--
-- Name: COLUMN lookup_uoc.model; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lookup_uoc.model IS 'The equipment model that correlates to the UOC';


--
-- Name: marine_corps_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marine_corps_management (
    id integer NOT NULL,
    niin character varying(255),
    sac character varying(255),
    cec character varying(255),
    mec character varying(255),
    mic character varying(255),
    otc character varying(255),
    pcc character varying(255)
);


--
-- Name: marine_corps_management_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.marine_corps_management_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: marine_corps_management_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.marine_corps_management_id_seq OWNED BY public.marine_corps_management.id;


--
-- Name: marines_mhif; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marines_mhif (
    niin character varying(255) NOT NULL,
    ui character varying(255),
    sac character varying(255),
    mec character varying(255),
    slc character varying(255),
    mc_rec_code character varying(255),
    ciic character varying(255),
    phrase_code character varying(255),
    pmic character varying(255),
    mc_cic character varying(255),
    pcc character varying(255),
    cec character varying(255),
    sub character varying(255),
    demil_code character varying(255),
    sos character varying(255),
    adpe_code character varying(255),
    aac character varying(255),
    mic character varying(255),
    mhif_date character varying(255),
    mc_item_name character varying(255),
    prime_fsc character varying(255),
    prime_niin character varying(255),
    mc_unit_price character varying(255)
);


--
-- Name: marines_sl_6_1; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marines_sl_6_1 (
    niin character varying(255) NOT NULL,
    idn character varying(255),
    type_model_number character varying(255),
    tam_number character varying(255),
    in_service_date character varying(255),
    act_sch character varying(255),
    exit_date character varying(255),
    spc character varying(255),
    tidc character varying(255),
    wsc character varying(255),
    cec character varying(255),
    lap character varying(255),
    alo character varying(255),
    mc_nomenclature character varying(255),
    repair_part_count character varying(255)
);


--
-- Name: marines_sl_6_2_item_id; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marines_sl_6_2_item_id (
    id integer NOT NULL,
    niin character varying(255),
    idn character varying(255),
    approved_item_name character varying(255),
    smr character varying(255),
    exit_date character varying(255),
    cec character varying(255)
);


--
-- Name: marines_sl_6_2_item_id_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.marines_sl_6_2_item_id_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: marines_sl_6_2_item_id_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.marines_sl_6_2_item_id_id_seq OWNED BY public.marines_sl_6_2_item_id.id;


--
-- Name: marines_sl_6_2_item_supp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marines_sl_6_2_item_supp (
    id integer NOT NULL,
    niin character varying(255),
    idn character varying(255),
    qty3 character varying(255),
    qty4 character varying(255),
    mc character varying(255),
    um character varying(255),
    ptrf character varying(255),
    wsc character varying(255),
    crit character varying(255)
);


--
-- Name: marines_sl_6_2_item_supp_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.marines_sl_6_2_item_supp_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: marines_sl_6_2_item_supp_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.marines_sl_6_2_item_supp_id_seq OWNED BY public.marines_sl_6_2_item_supp.id;


--
-- Name: marines_stock_list_6_3; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.marines_stock_list_6_3 (
    niin character varying(255) NOT NULL,
    ssr character varying(255),
    cm character varying(255),
    uur character varying(255),
    cei character varying(255),
    bii character varying(255),
    aal character varying(255),
    cli character varying(255),
    sl3_remarks character varying(255)
);


--
-- Name: material_images; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.material_images (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    niin character varying(9) NOT NULL,
    user_id text NOT NULL,
    blob_name text NOT NULL,
    blob_url text NOT NULL,
    original_filename text NOT NULL,
    file_size_bytes bigint NOT NULL,
    mime_type character varying(100) NOT NULL,
    upload_date timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    is_flagged boolean DEFAULT false NOT NULL,
    flag_count integer DEFAULT 0 NOT NULL,
    downvote_count integer DEFAULT 0 NOT NULL,
    upvote_count integer DEFAULT 0 NOT NULL,
    net_votes integer GENERATED ALWAYS AS ((upvote_count - downvote_count)) STORED,
    created_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT valid_niin CHECK ((length((niin)::text) = 9))
);


--
-- Name: material_images_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.material_images_flags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    image_id uuid NOT NULL,
    user_id text NOT NULL,
    reason character varying(50) NOT NULL,
    description text,
    created_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: material_images_upload_limits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.material_images_upload_limits (
    user_id text NOT NULL,
    niin character varying(9) NOT NULL,
    last_upload_time timestamp(6) without time zone NOT NULL,
    upload_count integer DEFAULT 1 NOT NULL
);


--
-- Name: material_images_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.material_images_votes (
    image_id uuid NOT NULL,
    user_id text NOT NULL,
    vote_type character varying(10) NOT NULL,
    created_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT material_images_votes_vote_type_check CHECK (((vote_type)::text = ANY (ARRAY[('upvote'::character varying)::text, ('downvote'::character varying)::text])))
);


--
-- Name: moe_rule; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.moe_rule (
    niin character varying(255) NOT NULL,
    moe_rl character varying(255) NOT NULL,
    moe_cd character varying(255),
    amc character varying(255),
    amsc character varying(255),
    nimsc character varying(255),
    dt_asgnd character varying(255),
    imc character varying(255),
    imca character varying(255),
    aac character varying(255),
    pica character varying(255),
    pica_loa character varying(255),
    sica character varying(255),
    sica_loa character varying(255),
    submtr character varying(255),
    auth_collab character varying(255),
    supp_collab character varying(255),
    auth_rcvr character varying(255),
    supp_rcvr character varying(255),
    dsor character varying(255),
    fmr_moe_rl character varying(255)
);


--
-- Name: navy_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.navy_management (
    niin character varying(255) NOT NULL,
    cog character varying(255),
    smic character varying(255),
    irrc character varying(255),
    smcc character varying(255)
);


--
-- Name: niin_lookup; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.niin_lookup AS
 SELECT nsn.niin,
    nsn.fsc,
    nsn.item_name,
    (army_master_data_file.niin IS NOT NULL) AS has_amdf,
    (flis_management_id.niin IS NOT NULL) AS has_flis
   FROM ((public.nsn
     LEFT JOIN public.army_master_data_file ON (((nsn.niin)::text = (army_master_data_file.niin)::text)))
     LEFT JOIN public.flis_management_id ON (((nsn.niin)::text = (flis_management_id.niin)::text)));


--
-- Name: part_number; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.part_number (
    niin character varying(255) NOT NULL,
    fsc character varying(255),
    item_name character varying(255),
    cage_code character varying(255) NOT NULL,
    company_name character varying(255),
    part_number character varying(255) NOT NULL,
    publication_date character varying(255)
);


--
-- Name: pol_products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pol_products (
    figure_header text NOT NULL,
    specification text NOT NULL,
    military_symbol text NOT NULL,
    container_size text NOT NULL,
    niin text NOT NULL
);


--
-- Name: ps_mag_summaries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ps_mag_summaries (
    file_name text NOT NULL,
    summary text
);


--
-- Name: quick_list_battery; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.quick_list_battery (
    nsn text NOT NULL,
    part_number text,
    model text,
    description text
);


--
-- Name: TABLE quick_list_battery; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.quick_list_battery IS 'Batteries for use in Quick Lists Battery Menu';


--
-- Name: quick_list_clothing; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.quick_list_clothing (
    nsn text NOT NULL,
    size text,
    description text
);


--
-- Name: quick_list_wheel_tires; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.quick_list_wheel_tires (
    id integer NOT NULL,
    vehicle text,
    assembly_nsn text,
    tire_nsn text,
    size text,
    item_comment text
);


--
-- Name: quick_list_wheel_tires_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.quick_list_wheel_tires_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


--
-- Name: quick_list_wheel_tires_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.quick_list_wheel_tires_id_seq OWNED BY public.quick_list_wheel_tires.id;


--
-- Name: sb_700_20_app_b; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_b (
    nsn text NOT NULL,
    lin text NOT NULL,
    reportable_item_cont text,
    chapter_code text
);


--
-- Name: sb_700_20_app_c; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_c (
    nomenclature text,
    lin text NOT NULL,
    chapter_code integer
);


--
-- Name: sb_700_20_app_d; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_d (
    lin text NOT NULL,
    nsn text NOT NULL,
    ric integer,
    chapter_code integer,
    type_of_action text
);


--
-- Name: sb_700_20_app_e; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_e (
    lin text NOT NULL,
    cmc text,
    reason_for_deletion text,
    new_lin text,
    nsn text NOT NULL,
    nomenclature text,
    date_entered_into_ap text,
    army_type_class text,
    ratio text,
    zmm_appdx_dlw_and_or_zmm text,
    calendar_year_month text,
    chapter_code integer
);


--
-- Name: sb_700_20_app_f; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_f (
    lin text NOT NULL,
    type_of_action text,
    chapter_code text
);


--
-- Name: sb_700_20_app_g; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_g (
    tr text,
    new_lin text,
    lin text NOT NULL
);


--
-- Name: sb_700_20_app_h1; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_h1 (
    lin_zmm_lin text NOT NULL,
    nomenclature text,
    lin_zmm_sublin text NOT NULL,
    sub_lin_nomenclature text
);


--
-- Name: sb_700_20_app_h2; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_h2 (
    lin_zmmsublin text NOT NULL,
    sub_lin_nomenclature text,
    lin_zmm_lin text NOT NULL,
    nomenclature text
);


--
-- Name: sb_700_20_app_i; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_i (
    lin text NOT NULL,
    chapter_code text
);


--
-- Name: sb_700_20_app_j; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_app_j (
    lin text NOT NULL,
    zlinum_pomcus text
);


--
-- Name: sb_700_20_chp_4; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_chp_4 (
    lin text NOT NULL,
    control_item_code text,
    reportable_item_cont_zmmlricc integer,
    nomenclature text,
    cmc text,
    ric text,
    current_mcn text,
    supply_catof_material text,
    reportable_item_cont_zmmnricc text,
    nsn_nomenclature text,
    standard_price text,
    unit_of_issue text,
    second_position_of_mara text,
    logistics_control_co text,
    army_type_class text,
    reference_data text
);


--
-- Name: sb_700_20_chp_6; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_chp_6 (
    lin text NOT NULL,
    control_item_code text,
    reportable_item_cont_zmmlricc text,
    nomenclature text,
    cmc text,
    ric text,
    current_mcn text NOT NULL,
    supply_catof_material text,
    reportable_item_cont_zmmnricc text,
    nsn_nomenclature text,
    standard_price text,
    unit_of_issue text,
    second_position_of_mara text,
    logistics_control_co text,
    army_type_class text,
    reference_data text
);


--
-- Name: sb_700_20_chp_8; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sb_700_20_chp_8 (
    lin text NOT NULL,
    control_item_code text,
    reportable_item_cont_zmmlricc text,
    nomenclature text,
    cmc text,
    ric text,
    current_mcn text NOT NULL,
    supply_catof_material text,
    reportable_item_cont_zmmnricc text,
    nsn_nomenclature text,
    standard_price text,
    unit_of_issue text,
    "second_position_ofMara" text,
    logistics_control_co text,
    army_type_class text,
    reference_data text
);


--
-- Name: shop_invite_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_invite_codes (
    id text NOT NULL,
    shop_id text NOT NULL,
    code text NOT NULL,
    created_by text NOT NULL,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: shop_list_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_list_items (
    id text NOT NULL,
    list_id text NOT NULL,
    niin text NOT NULL,
    nomenclature text NOT NULL,
    quantity integer NOT NULL,
    added_by text NOT NULL,
    created_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    updated_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    nickname text,
    unit_of_measure character varying(50)
);


--
-- Name: shop_lists; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_lists (
    id text NOT NULL,
    shop_id text NOT NULL,
    created_by text NOT NULL,
    description text NOT NULL,
    created_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    updated_at timestamp(6) with time zone DEFAULT now() NOT NULL
);


--
-- Name: shop_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_members (
    id text NOT NULL,
    shop_id text NOT NULL,
    user_id text NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    joined_at timestamp with time zone DEFAULT now(),
    CONSTRAINT shop_members_role_check CHECK ((role = ANY (ARRAY['member'::text, 'admin'::text])))
);


--
-- Name: shop_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_messages (
    id text NOT NULL,
    shop_id text NOT NULL,
    user_id text NOT NULL,
    message text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    is_edited boolean DEFAULT false,
    parent_id text
);


--
-- Name: shop_notification_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_notification_items (
    id text NOT NULL,
    shop_id text DEFAULT ''::text NOT NULL,
    notification_id text NOT NULL,
    niin text NOT NULL,
    nomenclature text NOT NULL,
    quantity integer DEFAULT 1 NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE shop_notification_items; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.shop_notification_items IS 'Items associated with vehicle notifications';


--
-- Name: COLUMN shop_notification_items.nomenclature; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_notification_items.nomenclature IS 'Official item name/description';


--
-- Name: shop_vehicle; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_vehicle (
    id text NOT NULL,
    creator_id text DEFAULT ''::text NOT NULL,
    niin text DEFAULT ''::text NOT NULL,
    admin text NOT NULL,
    model text DEFAULT ''::text NOT NULL,
    serial text DEFAULT ''::text NOT NULL,
    uoc text DEFAULT 'UNK'::text NOT NULL,
    mileage integer DEFAULT 0 NOT NULL,
    hours integer DEFAULT 0 NOT NULL,
    comment text DEFAULT ''::text NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL,
    last_updated timestamp(6) with time zone DEFAULT now() NOT NULL,
    shop_id text NOT NULL,
    tracked_mileage integer,
    tracked_hours integer,
    CONSTRAINT shop_vehicle_tracked_hours_nonnegative CHECK (((tracked_hours IS NULL) OR (tracked_hours >= 0))),
    CONSTRAINT shop_vehicle_tracked_mileage_nonnegative CHECK (((tracked_mileage IS NULL) OR (tracked_mileage >= 0)))
);


--
-- Name: TABLE shop_vehicle; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.shop_vehicle IS 'Vehicle information and tracking';


--
-- Name: COLUMN shop_vehicle.uoc; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_vehicle.uoc IS 'Usable On Code, default UNK for unknown';


--
-- Name: shop_vehicle_notification_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_vehicle_notification_changes (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    notification_id text,
    shop_id text NOT NULL,
    vehicle_id text,
    changed_by text,
    changed_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    change_type text NOT NULL,
    field_changes jsonb NOT NULL,
    notification_title text,
    notification_type text,
    vehicle_admin text
);


--
-- Name: COLUMN shop_vehicle_notification_changes.notification_title; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_vehicle_notification_changes.notification_title IS 'Denormalized: notification title at time of change. Preserved when notification is deleted.';


--
-- Name: COLUMN shop_vehicle_notification_changes.notification_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_vehicle_notification_changes.notification_type IS 'Denormalized: notification type at time of change. Preserved when notification is deleted.';


--
-- Name: COLUMN shop_vehicle_notification_changes.vehicle_admin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_vehicle_notification_changes.vehicle_admin IS 'Denormalized: vehicle admin number at time of change. Preserved when vehicle is deleted.';


--
-- Name: shop_vehicle_notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_vehicle_notifications (
    id text NOT NULL,
    shop_id text DEFAULT ''::text NOT NULL,
    vehicle_id text NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    type text NOT NULL,
    completed boolean DEFAULT false NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL,
    last_updated timestamp(6) with time zone DEFAULT now() NOT NULL,
    attached_shop_list text
);


--
-- Name: TABLE shop_vehicle_notifications; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.shop_vehicle_notifications IS 'Notifications related to vehicles (M1, PM, MW)';


--
-- Name: COLUMN shop_vehicle_notifications.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.shop_vehicle_notifications.type IS 'Type of notification: M1 (Maintenance), PM (Preventive Maintenance), MW (Modification)';


--
-- Name: shops; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shops (
    id text NOT NULL,
    name character varying(255) NOT NULL,
    details text DEFAULT ''::text,
    created_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    admin_only_lists boolean DEFAULT false NOT NULL
);


--
-- Name: socom_management; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.socom_management (
    niin character varying(255) NOT NULL,
    icp character varying(255),
    a_l character varying(255),
    rep character varying(255),
    w_esdc character varying(255),
    ar character varying(255),
    cos character varying(255)
);


--
-- Name: tmde_requirements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tmde_requirements (
    niin text NOT NULL,
    qty_skot integer,
    component text,
    "interval" text
);


--
-- Name: user_item_category; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_item_category (
    id text NOT NULL,
    user_uid text NOT NULL,
    name character varying(60) NOT NULL,
    comment character varying(255),
    image text,
    last_updated timestamp(6) without time zone
);


--
-- Name: user_item_comments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_item_comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_items_categorized; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_items_categorized (
    user_id text NOT NULL,
    niin text NOT NULL,
    item_name text,
    quantity integer,
    equip_model text,
    uoc text,
    category_id text NOT NULL,
    save_time timestamp(6) without time zone,
    image text,
    last_updated timestamp(6) without time zone,
    id text NOT NULL,
    nickname character varying(50)
);


--
-- Name: user_items_quick; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_items_quick (
    user_id text NOT NULL,
    niin text NOT NULL,
    item_name text,
    item_comment text,
    save_time timestamp(6) without time zone,
    last_updated timestamp(6) without time zone,
    id text NOT NULL,
    image text,
    nickname character varying(50)
);


--
-- Name: user_items_serialized; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_items_serialized (
    user_id text NOT NULL,
    niin text NOT NULL,
    item_name text,
    serial text NOT NULL,
    image text,
    save_time timestamp(6) without time zone,
    item_comment text,
    last_updated timestamp(6) without time zone,
    id text NOT NULL,
    nickname character varying(50)
);


--
-- Name: user_notification_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_notification_items (
    id text NOT NULL,
    user_id text DEFAULT ''::text NOT NULL,
    notification_id text NOT NULL,
    niin text NOT NULL,
    nomenclature text NOT NULL,
    quantity integer DEFAULT 1 NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE user_notification_items; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.user_notification_items IS 'Items associated with vehicle notifications';


--
-- Name: COLUMN user_notification_items.nomenclature; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.user_notification_items.nomenclature IS 'Official item name/description';


--
-- Name: user_pmcs_checklists; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_checklists (
    id uuid NOT NULL,
    owner_uid text,
    sync_version bigint NOT NULL,
    account_change_version bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT user_pmcs_checklists_account_change_version_check CHECK ((account_change_version > 0)),
    CONSTRAINT user_pmcs_checklists_owner_state_check CHECK (((owner_uid IS NOT NULL) OR (deleted_at IS NOT NULL))),
    CONSTRAINT user_pmcs_checklists_sync_version_check CHECK ((sync_version > 0))
);


--
-- Name: user_pmcs_community_releases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_community_releases (
    revision_id uuid NOT NULL,
    checklist_id uuid NOT NULL,
    released_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_pmcs_community_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_community_sources (
    checklist_id uuid NOT NULL,
    status text NOT NULL,
    current_release_revision_id uuid,
    latest_release_revision_number integer NOT NULL,
    first_released_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    retired_at timestamp with time zone,
    CONSTRAINT user_pmcs_community_sources_latest_release_revision_numbe_check CHECK ((latest_release_revision_number > 0)),
    CONSTRAINT user_pmcs_community_sources_state_check CHECK ((((status = 'active'::text) AND (current_release_revision_id IS NOT NULL) AND (retired_at IS NULL)) OR ((status = 'retired'::text) AND (current_release_revision_id IS NULL) AND (retired_at IS NOT NULL)))),
    CONSTRAINT user_pmcs_community_sources_status_check CHECK ((status = ANY (ARRAY['active'::text, 'retired'::text])))
);


--
-- Name: user_pmcs_community_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_community_votes (
    checklist_id uuid NOT NULL,
    voter_uid text NOT NULL,
    direction smallint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_pmcs_community_votes_direction_check CHECK ((direction = ANY (ARRAY['-1'::integer, 1])))
);


--
-- Name: user_pmcs_content_uuid_reservations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_content_uuid_reservations (
    id uuid NOT NULL
);


--
-- Name: user_pmcs_faults; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_faults (
    pmcs_id uuid NOT NULL,
    section_id text NOT NULL,
    item_index integer NOT NULL,
    item_no text NOT NULL,
    status text NOT NULL,
    fault_text text NOT NULL,
    corrective_action text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    section_title text,
    CONSTRAINT user_pmcs_faults_item_index_check CHECK ((item_index >= 0)),
    CONSTRAINT user_pmcs_faults_nonblank_fields_check CHECK (((btrim(section_id) <> ''::text) AND (btrim(item_no) <> ''::text) AND (btrim(fault_text) <> ''::text))),
    CONSTRAINT user_pmcs_faults_status_check CHECK ((status = ANY (ARRAY['x'::text, 'slash'::text, 'dash'::text])))
);


--
-- Name: user_pmcs_inspection_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_inspection_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pmcs_id uuid NOT NULL,
    author_id text NOT NULL,
    text text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT user_pmcs_inspection_comments_nonblank_check CHECK ((btrim(text) <> ''::text))
);


--
-- Name: user_pmcs_inspections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_inspections (
    id uuid NOT NULL,
    equipment_id text NOT NULL,
    guide_manual text,
    performed_date timestamp with time zone NOT NULL,
    performed_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    source_type text NOT NULL,
    custom_checklist_id uuid,
    custom_revision_id uuid,
    custom_revision_number integer,
    custom_checklist_name text,
    CONSTRAINT user_pmcs_inspections_equipment_id_nonblank_check CHECK ((btrim(equipment_id) <> ''::text)),
    CONSTRAINT user_pmcs_inspections_source_shape_check CHECK ((((source_type = 'guide'::text) AND (guide_manual IS NOT NULL) AND (guide_manual = btrim(guide_manual)) AND (guide_manual ~~ 'pmcs_sbs/%'::text) AND ("right"(guide_manual, 5) = '.json'::text) AND (custom_checklist_id IS NULL) AND (custom_revision_id IS NULL) AND (custom_revision_number IS NULL) AND (custom_checklist_name IS NULL)) OR ((source_type = 'custom'::text) AND (guide_manual IS NULL) AND (custom_checklist_id IS NOT NULL) AND (custom_revision_id IS NOT NULL) AND (custom_revision_number IS NOT NULL) AND (custom_revision_number >= 0) AND (custom_checklist_name IS NOT NULL) AND (custom_checklist_name = btrim(custom_checklist_name)) AND (btrim(custom_checklist_name) <> ''::text)))),
    CONSTRAINT user_pmcs_inspections_source_type_check CHECK ((source_type = ANY (ARRAY['guide'::text, 'custom'::text])))
);


--
-- Name: user_pmcs_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_items (
    id uuid NOT NULL,
    section_id uuid NOT NULL,
    "position" integer NOT NULL,
    "interval" text DEFAULT ''::text NOT NULL,
    item_to_be_checked_or_serviced text DEFAULT ''::text NOT NULL,
    performed_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT user_pmcs_items_position_check CHECK (("position" > 0)),
    CONSTRAINT user_pmcs_items_text_bytes_check CHECK (((octet_length("interval") <= 8192) AND (octet_length(performed_by) <= 8192) AND (octet_length(item_to_be_checked_or_serviced) <= 65536)))
);


--
-- Name: user_pmcs_notices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_notices (
    id uuid NOT NULL,
    item_id uuid NOT NULL,
    "position" integer NOT NULL,
    type text,
    notice_text text DEFAULT ''::text NOT NULL,
    CONSTRAINT user_pmcs_notices_position_check CHECK (("position" > 0)),
    CONSTRAINT user_pmcs_notices_text_bytes_check CHECK ((octet_length(notice_text) <= 65536)),
    CONSTRAINT user_pmcs_notices_type_check CHECK ((type = ANY (ARRAY['warning'::text, 'caution'::text, 'note'::text])))
);


--
-- Name: user_pmcs_procedure_steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_procedure_steps (
    id uuid NOT NULL,
    item_id uuid NOT NULL,
    "position" integer NOT NULL,
    step_text text DEFAULT ''::text NOT NULL,
    fault_found_if text DEFAULT ''::text NOT NULL,
    CONSTRAINT user_pmcs_procedure_steps_position_check CHECK (("position" > 0)),
    CONSTRAINT user_pmcs_procedure_steps_text_bytes_check CHECK (((octet_length(step_text) <= 65536) AND (octet_length(fault_found_if) <= 65536)))
);


--
-- Name: user_pmcs_revision_models; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_revision_models (
    revision_id uuid NOT NULL,
    display_text text NOT NULL,
    normalized_text text NOT NULL,
    CONSTRAINT user_pmcs_revision_models_bytes_check CHECK (((octet_length(display_text) <= 8192) AND (octet_length(normalized_text) <= 8192)))
);


--
-- Name: user_pmcs_revisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_revisions (
    id uuid NOT NULL,
    checklist_id uuid NOT NULL,
    state text NOT NULL,
    revision_number integer,
    name text DEFAULT ''::text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    content_hash bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    CONSTRAINT user_pmcs_revisions_hash_check CHECK ((octet_length(content_hash) = 32)),
    CONSTRAINT user_pmcs_revisions_state_check CHECK ((((state = 'draft'::text) AND (revision_number IS NULL) AND (published_at IS NULL)) OR ((state = ANY (ARRAY['published'::text, 'superseded'::text])) AND (revision_number > 0) AND (published_at IS NOT NULL)))),
    CONSTRAINT user_pmcs_revisions_text_bytes_check CHECK (((octet_length(name) <= 8192) AND (octet_length(description) <= 65536)))
);


--
-- Name: user_pmcs_section_models; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_section_models (
    section_id uuid NOT NULL,
    display_text text NOT NULL,
    normalized_text text NOT NULL,
    CONSTRAINT user_pmcs_section_models_bytes_check CHECK (((octet_length(display_text) <= 8192) AND (octet_length(normalized_text) <= 8192)))
);


--
-- Name: user_pmcs_sections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_sections (
    id uuid NOT NULL,
    revision_id uuid NOT NULL,
    "position" integer NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    CONSTRAINT user_pmcs_sections_position_check CHECK (("position" > 0)),
    CONSTRAINT user_pmcs_sections_title_bytes_check CHECK ((octet_length(title) <= 8192))
);


--
-- Name: user_pmcs_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_subscriptions (
    subscriber_uid text NOT NULL,
    checklist_id uuid NOT NULL,
    installed_revision_id uuid,
    sync_version bigint NOT NULL,
    account_change_version bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT user_pmcs_subscriptions_account_change_version_check CHECK ((account_change_version > 0)),
    CONSTRAINT user_pmcs_subscriptions_state_check CHECK ((((deleted_at IS NULL) AND (installed_revision_id IS NOT NULL)) OR ((deleted_at IS NOT NULL) AND (installed_revision_id IS NULL)))),
    CONSTRAINT user_pmcs_subscriptions_sync_version_check CHECK ((sync_version > 0))
);


--
-- Name: user_pmcs_sync_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pmcs_sync_state (
    user_uid text NOT NULL,
    current_version bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_pmcs_sync_state_current_version_check CHECK ((current_version >= 0))
);


--
-- Name: user_suggestion_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_suggestion_votes (
    suggestion_id uuid NOT NULL,
    voter_id text NOT NULL,
    direction smallint NOT NULL,
    created_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_suggestion_votes_direction_check CHECK ((direction = ANY (ARRAY['-1'::integer, 1])))
);


--
-- Name: user_suggestions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_suggestions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id text NOT NULL,
    title text NOT NULL,
    description text NOT NULL,
    status text DEFAULT 'Submitted'::text NOT NULL,
    created_at timestamp(6) with time zone DEFAULT now() NOT NULL,
    updated_at timestamp(6) with time zone,
    show boolean,
    CONSTRAINT user_suggestions_description_check CHECK (((length(description) >= 1) AND (length(description) <= 2000))),
    CONSTRAINT user_suggestions_title_check CHECK (((length(title) >= 1) AND (length(title) <= 200)))
);


--
-- Name: user_vehicle; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_vehicle (
    id text NOT NULL,
    user_id text DEFAULT ''::text NOT NULL,
    niin text DEFAULT ''::text NOT NULL,
    admin text NOT NULL,
    model text DEFAULT ''::text NOT NULL,
    serial text DEFAULT ''::text NOT NULL,
    uoc text DEFAULT 'UNK'::text NOT NULL,
    mileage integer DEFAULT 0 NOT NULL,
    hours integer DEFAULT 0 NOT NULL,
    comment text DEFAULT ''::text NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL,
    last_updated timestamp(6) with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE user_vehicle; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.user_vehicle IS 'Vehicle information and tracking';


--
-- Name: COLUMN user_vehicle.uoc; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.user_vehicle.uoc IS 'Usable On Code, default UNK for unknown';


--
-- Name: user_vehicle_notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_vehicle_notifications (
    id text NOT NULL,
    user_id text DEFAULT ''::text NOT NULL,
    vehicle_id text NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    type text NOT NULL,
    completed boolean DEFAULT false NOT NULL,
    save_time timestamp(6) with time zone DEFAULT now() NOT NULL,
    last_updated timestamp(6) with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE user_vehicle_notifications; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.user_vehicle_notifications IS 'Notifications related to vehicles (M1, PM, MW)';


--
-- Name: COLUMN user_vehicle_notifications.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.user_vehicle_notifications.type IS 'Type of notification: M1 (Maintenance), PM (Preventive Maintenance), MW (Modification)';


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    uid text NOT NULL,
    email character varying(70) NOT NULL,
    username character varying(25) DEFAULT 'Anonymous'::character varying,
    created_at timestamp(6) without time zone DEFAULT now(),
    is_enabled boolean DEFAULT true,
    last_login timestamp(6) without time zone DEFAULT now()
);


--
-- Name: amdf_i_and_s id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_i_and_s ALTER COLUMN id SET DEFAULT nextval('public.amdf_i_and_s_id_seq'::regclass);


--
-- Name: amdf_phrase id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_phrase ALTER COLUMN id SET DEFAULT nextval('public.amdf_phrase_id_seq'::regclass);


--
-- Name: army_management id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_management ALTER COLUMN id SET DEFAULT nextval('public.army_management_id_seq'::regclass);


--
-- Name: faa_management id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faa_management ALTER COLUMN id SET DEFAULT nextval('public.faa_management_id_seq'::regclass);


--
-- Name: flis_cancelled_niin id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_cancelled_niin ALTER COLUMN id SET DEFAULT nextval('public.flis_cancelled_niin_id_seq'::regclass);


--
-- Name: flis_item_characteristics id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_item_characteristics ALTER COLUMN id SET DEFAULT nextval('public.flis_item_characteristics_id_seq'::regclass);


--
-- Name: flis_management id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_management ALTER COLUMN id SET DEFAULT nextval('public.flis_management_id_seq'::regclass);


--
-- Name: flis_packaging_1 id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_packaging_1 ALTER COLUMN id SET DEFAULT nextval('public.flis_packaging_1_id_seq'::regclass);


--
-- Name: flis_packaging_2 id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_packaging_2 ALTER COLUMN id SET DEFAULT nextval('public.flis_packaging_2_id_seq'::regclass);


--
-- Name: flis_phrase id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_phrase ALTER COLUMN id SET DEFAULT nextval('public.flis_phrase_id_seq'::regclass);


--
-- Name: flis_standardization id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_standardization ALTER COLUMN id SET DEFAULT nextval('public.flis_standardization_id_seq'::regclass);


--
-- Name: marine_corps_management id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marine_corps_management ALTER COLUMN id SET DEFAULT nextval('public.marine_corps_management_id_seq'::regclass);


--
-- Name: marines_sl_6_2_item_id id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_sl_6_2_item_id ALTER COLUMN id SET DEFAULT nextval('public.marines_sl_6_2_item_id_id_seq'::regclass);


--
-- Name: marines_sl_6_2_item_supp id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_sl_6_2_item_supp ALTER COLUMN id SET DEFAULT nextval('public.marines_sl_6_2_item_supp_id_seq'::regclass);


--
-- Name: quick_list_wheel_tires id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.quick_list_wheel_tires ALTER COLUMN id SET DEFAULT nextval('public.quick_list_wheel_tires_id_seq'::regclass);


--
-- Name: air_force_management air_force_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.air_force_management
    ADD CONSTRAINT air_force_management_pkey PRIMARY KEY (niin);


--
-- Name: amdf_billing amdf_billing_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_billing
    ADD CONSTRAINT amdf_billing_pkey PRIMARY KEY (niin);


--
-- Name: amdf_credit amdf_credit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_credit
    ADD CONSTRAINT amdf_credit_pkey PRIMARY KEY (niin);


--
-- Name: amdf_freight amdf_freight_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_freight
    ADD CONSTRAINT amdf_freight_pkey PRIMARY KEY (niin);


--
-- Name: amdf_i_and_s amdf_i_and_s_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_i_and_s
    ADD CONSTRAINT amdf_i_and_s_pkey PRIMARY KEY (id);


--
-- Name: amdf_management amdf_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_management
    ADD CONSTRAINT amdf_management_pkey PRIMARY KEY (niin);


--
-- Name: amdf_matcat amdf_matcat_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_matcat
    ADD CONSTRAINT amdf_matcat_pkey PRIMARY KEY (niin);


--
-- Name: amdf_phrase amdf_phrase_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.amdf_phrase
    ADD CONSTRAINT amdf_phrase_pkey PRIMARY KEY (id);


--
-- Name: analytics_event_counters analytics_event_counters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytics_event_counters
    ADD CONSTRAINT analytics_event_counters_pkey PRIMARY KEY (id);


--
-- Name: army_freight army_freight_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_freight
    ADD CONSTRAINT army_freight_pkey PRIMARY KEY (niin);


--
-- Name: army_lin_to_niin army_lin_to_niin_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_lin_to_niin
    ADD CONSTRAINT army_lin_to_niin_pkey PRIMARY KEY (niin);


--
-- Name: army_line_item_number army_line_item_number_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_line_item_number
    ADD CONSTRAINT army_line_item_number_pkey PRIMARY KEY (lin);


--
-- Name: army_management army_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_management
    ADD CONSTRAINT army_management_pkey PRIMARY KEY (id);


--
-- Name: army_master_data_file army_master_data_file_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_master_data_file
    ADD CONSTRAINT army_master_data_file_pkey PRIMARY KEY (niin);


--
-- Name: army_pack_supplemental_instruct army_pack_supplemental_instruct_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_pack_supplemental_instruct
    ADD CONSTRAINT army_pack_supplemental_instruct_pkey PRIMARY KEY (niin);


--
-- Name: army_packaging_1 army_packaging_1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_packaging_1
    ADD CONSTRAINT army_packaging_1_pkey PRIMARY KEY (niin);


--
-- Name: army_packaging_2 army_packaging_2_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_packaging_2
    ADD CONSTRAINT army_packaging_2_pkey PRIMARY KEY (niin);


--
-- Name: army_packaging_and_freight army_packaging_and_freight_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_packaging_and_freight
    ADD CONSTRAINT army_packaging_and_freight_pkey PRIMARY KEY (niin);


--
-- Name: army_packaging_special_instruct army_packaging_special_instruct_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_packaging_special_instruct
    ADD CONSTRAINT army_packaging_special_instruct_pkey PRIMARY KEY (niin);


--
-- Name: army_related_nsn army_related_nsn_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_related_nsn
    ADD CONSTRAINT army_related_nsn_pkey PRIMARY KEY (niin);


--
-- Name: army_sarsscat army_sarsscat_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_sarsscat
    ADD CONSTRAINT army_sarsscat_pkey PRIMARY KEY (niin);


--
-- Name: army_substitute_lin army_substitute_lin_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.army_substitute_lin
    ADD CONSTRAINT army_substitute_lin_pkey PRIMARY KEY (lin, substitute_lin);


--
-- Name: cage_address cage_address_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cage_address
    ADD CONSTRAINT cage_address_pkey PRIMARY KEY (cage_code);


--
-- Name: cage_status_and_type cage_status_and_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cage_status_and_type
    ADD CONSTRAINT cage_status_and_type_pkey PRIMARY KEY (cage_code);


--
-- Name: coast_guard_management coast_guard_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.coast_guard_management
    ADD CONSTRAINT coast_guard_management_pkey PRIMARY KEY (niin);


--
-- Name: colloquial_name colloquial_name_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.colloquial_name
    ADD CONSTRAINT colloquial_name_pkey PRIMARY KEY (inc, colloquial_name, related_inc);


--
-- Name: component_end_item component_end_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.component_end_item
    ADD CONSTRAINT component_end_item_pkey PRIMARY KEY (niin, wpn_sys_id, wpn_sys_svc);


--
-- Name: disposition disposition_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.disposition
    ADD CONSTRAINT disposition_pkey PRIMARY KEY (niin);


--
-- Name: docs_equipment_details docs_equipment_details_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.docs_equipment_details
    ADD CONSTRAINT docs_equipment_details_pkey PRIMARY KEY (id);


--
-- Name: dss_weight_and_cube dss_weight_and_cube_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dss_weight_and_cube
    ADD CONSTRAINT dss_weight_and_cube_pkey PRIMARY KEY (niin);


--
-- Name: eic eic_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.eic
    ADD CONSTRAINT eic_pkey PRIMARY KEY (niin, mrc, uoeic);


--
-- Name: equipment_services equipment_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equipment_services
    ADD CONSTRAINT equipment_services_pkey PRIMARY KEY (id);


--
-- Name: faa_management faa_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faa_management
    ADD CONSTRAINT faa_management_pkey PRIMARY KEY (id);


--
-- Name: flis_cancelled_niin flis_cancelled_niin_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_cancelled_niin
    ADD CONSTRAINT flis_cancelled_niin_pkey PRIMARY KEY (id);


--
-- Name: flis_freight flis_freight_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_freight
    ADD CONSTRAINT flis_freight_pkey PRIMARY KEY (niin);


--
-- Name: flis_identification flis_identification_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_identification
    ADD CONSTRAINT flis_identification_pkey PRIMARY KEY (niin);


--
-- Name: flis_item_characteristics flis_item_characteristics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_item_characteristics
    ADD CONSTRAINT flis_item_characteristics_pkey PRIMARY KEY (id);


--
-- Name: flis_management_id flis_management_id_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_management_id
    ADD CONSTRAINT flis_management_id_pkey PRIMARY KEY (niin);


--
-- Name: flis_management flis_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_management
    ADD CONSTRAINT flis_management_pkey PRIMARY KEY (id);


--
-- Name: flis_packaging_1 flis_packaging_1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_packaging_1
    ADD CONSTRAINT flis_packaging_1_pkey PRIMARY KEY (id);


--
-- Name: flis_packaging_2 flis_packaging_2_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_packaging_2
    ADD CONSTRAINT flis_packaging_2_pkey PRIMARY KEY (id);


--
-- Name: flis_phrase flis_phrase_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_phrase
    ADD CONSTRAINT flis_phrase_pkey PRIMARY KEY (id);


--
-- Name: flis_reference flis_reference_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_reference
    ADD CONSTRAINT flis_reference_pkey PRIMARY KEY (niin, part_number, cage_code);


--
-- Name: flis_standardization flis_standardization_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flis_standardization
    ADD CONSTRAINT flis_standardization_pkey PRIMARY KEY (id);


--
-- Name: help help_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.help
    ADD CONSTRAINT help_pkey PRIMARY KEY (code, description);


--
-- Name: item_comment_flags item_comment_flags__uq_comment_flagger; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comment_flags
    ADD CONSTRAINT item_comment_flags__uq_comment_flagger UNIQUE (comment_id, flagger_id);


--
-- Name: item_comment_flags item_comment_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comment_flags
    ADD CONSTRAINT item_comment_flags_pkey PRIMARY KEY (id);


--
-- Name: item_comments item_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comments
    ADD CONSTRAINT item_comments_pkey PRIMARY KEY (id);


--
-- Name: marine_corps_management marine_corps_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marine_corps_management
    ADD CONSTRAINT marine_corps_management_pkey PRIMARY KEY (id);


--
-- Name: marines_mhif marines_mhif_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_mhif
    ADD CONSTRAINT marines_mhif_pkey PRIMARY KEY (niin);


--
-- Name: marines_sl_6_1 marines_sl_6_1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_sl_6_1
    ADD CONSTRAINT marines_sl_6_1_pkey PRIMARY KEY (niin);


--
-- Name: marines_sl_6_2_item_id marines_sl_6_2_item_id_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_sl_6_2_item_id
    ADD CONSTRAINT marines_sl_6_2_item_id_pkey PRIMARY KEY (id);


--
-- Name: marines_sl_6_2_item_supp marines_sl_6_2_item_supp_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_sl_6_2_item_supp
    ADD CONSTRAINT marines_sl_6_2_item_supp_pkey PRIMARY KEY (id);


--
-- Name: marines_stock_list_6_3 marines_stock_list_6_3_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.marines_stock_list_6_3
    ADD CONSTRAINT marines_stock_list_6_3_pkey PRIMARY KEY (niin);


--
-- Name: material_images material_images_blob_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images
    ADD CONSTRAINT material_images_blob_name_key UNIQUE (blob_name);


--
-- Name: material_images_flags material_images_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_flags
    ADD CONSTRAINT material_images_flags_pkey PRIMARY KEY (id);


--
-- Name: material_images material_images_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images
    ADD CONSTRAINT material_images_pkey PRIMARY KEY (id);


--
-- Name: material_images_upload_limits material_images_upload_limits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_upload_limits
    ADD CONSTRAINT material_images_upload_limits_pkey PRIMARY KEY (user_id, niin);


--
-- Name: material_images_votes material_images_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_votes
    ADD CONSTRAINT material_images_votes_pkey PRIMARY KEY (image_id, user_id);


--
-- Name: moe_rule moe_rule_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.moe_rule
    ADD CONSTRAINT moe_rule_pkey PRIMARY KEY (niin, moe_rl);


--
-- Name: navy_management navy_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.navy_management
    ADD CONSTRAINT navy_management_pkey PRIMARY KEY (niin);


--
-- Name: nsn nsn_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nsn
    ADD CONSTRAINT nsn_pkey PRIMARY KEY (niin);


--
-- Name: part_number part_number_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.part_number
    ADD CONSTRAINT part_number_pkey PRIMARY KEY (niin, part_number, cage_code);


--
-- Name: pol_products pol_products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pol_products
    ADD CONSTRAINT pol_products_pkey PRIMARY KEY (specification, niin);


--
-- Name: ps_mag_summaries ps_mag_summaries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ps_mag_summaries
    ADD CONSTRAINT ps_mag_summaries_pkey PRIMARY KEY (file_name);


--
-- Name: quick_list_battery quick_list_battery_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.quick_list_battery
    ADD CONSTRAINT quick_list_battery_pk PRIMARY KEY (nsn);


--
-- Name: quick_list_clothing quick_list_clothing_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.quick_list_clothing
    ADD CONSTRAINT quick_list_clothing_pk PRIMARY KEY (nsn);


--
-- Name: quick_list_wheel_tires quick_list_wheel_tires_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.quick_list_wheel_tires
    ADD CONSTRAINT quick_list_wheel_tires_pk PRIMARY KEY (id);


--
-- Name: sb_700_20_app_b sb_700_20_app_b_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_b
    ADD CONSTRAINT sb_700_20_app_b_pkey PRIMARY KEY (nsn, lin);


--
-- Name: sb_700_20_app_c sb_700_20_app_c_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_c
    ADD CONSTRAINT sb_700_20_app_c_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_app_d sb_700_20_app_d_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_d
    ADD CONSTRAINT sb_700_20_app_d_pkey PRIMARY KEY (lin, nsn);


--
-- Name: sb_700_20_app_e sb_700_20_app_e_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_e
    ADD CONSTRAINT sb_700_20_app_e_pkey PRIMARY KEY (lin, nsn);


--
-- Name: sb_700_20_app_f sb_700_20_app_f_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_f
    ADD CONSTRAINT sb_700_20_app_f_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_app_g sb_700_20_app_g_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_g
    ADD CONSTRAINT sb_700_20_app_g_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_app_h1 sb_700_20_app_h1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_h1
    ADD CONSTRAINT sb_700_20_app_h1_pkey PRIMARY KEY (lin_zmm_lin, lin_zmm_sublin);


--
-- Name: sb_700_20_app_h2 sb_700_20_app_h2_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_h2
    ADD CONSTRAINT sb_700_20_app_h2_pkey PRIMARY KEY (lin_zmmsublin, lin_zmm_lin);


--
-- Name: sb_700_20_app_i sb_700_20_app_i_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_i
    ADD CONSTRAINT sb_700_20_app_i_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_app_j sb_700_20_app_j_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_app_j
    ADD CONSTRAINT sb_700_20_app_j_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_chp_4 sb_700_20_chp_4_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_chp_4
    ADD CONSTRAINT sb_700_20_chp_4_pkey PRIMARY KEY (lin);


--
-- Name: sb_700_20_chp_6 sb_700_20_chp_6_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_chp_6
    ADD CONSTRAINT sb_700_20_chp_6_pkey PRIMARY KEY (lin, current_mcn);


--
-- Name: sb_700_20_chp_8 sb_700_20_chp_8_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sb_700_20_chp_8
    ADD CONSTRAINT sb_700_20_chp_8_pkey PRIMARY KEY (lin, current_mcn);


--
-- Name: shop_invite_codes shop_invite_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_invite_codes
    ADD CONSTRAINT shop_invite_codes_code_key UNIQUE (code);


--
-- Name: shop_invite_codes shop_invite_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_invite_codes
    ADD CONSTRAINT shop_invite_codes_pkey PRIMARY KEY (id);


--
-- Name: shop_list_items shop_list_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_list_items
    ADD CONSTRAINT shop_list_items_pkey PRIMARY KEY (id);


--
-- Name: shop_lists shop_lists_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_lists
    ADD CONSTRAINT shop_lists_pkey PRIMARY KEY (id);


--
-- Name: shop_members shop_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_members
    ADD CONSTRAINT shop_members_pkey PRIMARY KEY (id);


--
-- Name: shop_members shop_members_shop_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_members
    ADD CONSTRAINT shop_members_shop_id_user_id_key UNIQUE (shop_id, user_id);


--
-- Name: shop_messages shop_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_messages
    ADD CONSTRAINT shop_messages_pkey PRIMARY KEY (id);


--
-- Name: shop_notification_items shop_notification_items_niin_shop_id_notification_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_notification_items
    ADD CONSTRAINT shop_notification_items_niin_shop_id_notification_id_key UNIQUE (niin, shop_id, notification_id);


--
-- Name: shop_vehicle shop_vehicle_admin_serial_shop_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle
    ADD CONSTRAINT shop_vehicle_admin_serial_shop_key UNIQUE (admin, serial, shop_id);


--
-- Name: shop_vehicle_notification_changes shop_vehicle_notification_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notification_changes
    ADD CONSTRAINT shop_vehicle_notification_changes_pkey PRIMARY KEY (id);


--
-- Name: shops shops_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shops
    ADD CONSTRAINT shops_pkey PRIMARY KEY (id);


--
-- Name: socom_management socom_management_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.socom_management
    ADD CONSTRAINT socom_management_pkey PRIMARY KEY (niin);


--
-- Name: tmde_requirements tmde_requirements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tmde_requirements
    ADD CONSTRAINT tmde_requirements_pkey PRIMARY KEY (niin);


--
-- Name: user_vehicle unique_admin_serial; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_vehicle
    ADD CONSTRAINT unique_admin_serial UNIQUE (admin, serial);


--
-- Name: user_notification_items unique_niin_user_notif; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_items
    ADD CONSTRAINT unique_niin_user_notif UNIQUE (niin, user_id, notification_id);


--
-- Name: material_images_flags unique_user_image_flag; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_flags
    ADD CONSTRAINT unique_user_image_flag UNIQUE (image_id, user_id);


--
-- Name: users unique_users_email; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT unique_users_email UNIQUE (email);


--
-- Name: CONSTRAINT unique_users_email ON users; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT unique_users_email ON public.users IS 'Ensure only a single email is used';


--
-- Name: lookup_uoc usable_on_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lookup_uoc
    ADD CONSTRAINT usable_on_codes_pkey PRIMARY KEY (uoc, model);


--
-- Name: user_item_category user_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_item_category
    ADD CONSTRAINT user_categories_pkey PRIMARY KEY (id, user_uid);


--
-- Name: user_items_categorized user_items_category_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_categorized
    ADD CONSTRAINT user_items_category_pk PRIMARY KEY (niin, category_id, id);


--
-- Name: user_items_quick user_items_quick_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_quick
    ADD CONSTRAINT user_items_quick_pkey PRIMARY KEY (user_id, niin, id);


--
-- Name: user_items_serialized user_items_serialized_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_serialized
    ADD CONSTRAINT user_items_serialized_pk PRIMARY KEY (user_id, niin, serial, id);


--
-- Name: shop_notification_items user_notification_items_copy1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_notification_items
    ADD CONSTRAINT user_notification_items_copy1_pkey PRIMARY KEY (id);


--
-- Name: user_notification_items user_notification_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_items
    ADD CONSTRAINT user_notification_items_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_checklists user_pmcs_checklists_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_checklists
    ADD CONSTRAINT user_pmcs_checklists_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_community_releases user_pmcs_community_releases_checklist_revision_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_releases
    ADD CONSTRAINT user_pmcs_community_releases_checklist_revision_key UNIQUE (checklist_id, revision_id);


--
-- Name: user_pmcs_community_releases user_pmcs_community_releases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_releases
    ADD CONSTRAINT user_pmcs_community_releases_pkey PRIMARY KEY (revision_id);


--
-- Name: user_pmcs_community_sources user_pmcs_community_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_sources
    ADD CONSTRAINT user_pmcs_community_sources_pkey PRIMARY KEY (checklist_id);


--
-- Name: user_pmcs_community_votes user_pmcs_community_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_votes
    ADD CONSTRAINT user_pmcs_community_votes_pkey PRIMARY KEY (checklist_id, voter_uid);


--
-- Name: user_pmcs_content_uuid_reservations user_pmcs_content_uuid_reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_content_uuid_reservations
    ADD CONSTRAINT user_pmcs_content_uuid_reservations_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_faults user_pmcs_faults_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_faults
    ADD CONSTRAINT user_pmcs_faults_pkey PRIMARY KEY (pmcs_id, section_id, item_index);


--
-- Name: user_pmcs_inspection_comments user_pmcs_inspection_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspection_comments
    ADD CONSTRAINT user_pmcs_inspection_comments_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_inspections user_pmcs_inspections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspections
    ADD CONSTRAINT user_pmcs_inspections_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_items user_pmcs_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_items
    ADD CONSTRAINT user_pmcs_items_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_items user_pmcs_items_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_items
    ADD CONSTRAINT user_pmcs_items_position_key UNIQUE (section_id, "position");


--
-- Name: user_pmcs_notices user_pmcs_notices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_notices
    ADD CONSTRAINT user_pmcs_notices_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_notices user_pmcs_notices_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_notices
    ADD CONSTRAINT user_pmcs_notices_position_key UNIQUE (item_id, "position");


--
-- Name: user_pmcs_procedure_steps user_pmcs_procedure_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_procedure_steps
    ADD CONSTRAINT user_pmcs_procedure_steps_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_procedure_steps user_pmcs_procedure_steps_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_procedure_steps
    ADD CONSTRAINT user_pmcs_procedure_steps_position_key UNIQUE (item_id, "position");


--
-- Name: user_pmcs_revision_models user_pmcs_revision_models_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_revision_models
    ADD CONSTRAINT user_pmcs_revision_models_pkey PRIMARY KEY (revision_id, normalized_text);


--
-- Name: user_pmcs_revisions user_pmcs_revisions_checklist_id_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_revisions
    ADD CONSTRAINT user_pmcs_revisions_checklist_id_id_key UNIQUE (checklist_id, id);


--
-- Name: user_pmcs_revisions user_pmcs_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_revisions
    ADD CONSTRAINT user_pmcs_revisions_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_section_models user_pmcs_section_models_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_section_models
    ADD CONSTRAINT user_pmcs_section_models_pkey PRIMARY KEY (section_id, normalized_text);


--
-- Name: user_pmcs_sections user_pmcs_sections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_sections
    ADD CONSTRAINT user_pmcs_sections_pkey PRIMARY KEY (id);


--
-- Name: user_pmcs_sections user_pmcs_sections_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_sections
    ADD CONSTRAINT user_pmcs_sections_position_key UNIQUE (revision_id, "position");


--
-- Name: user_pmcs_subscriptions user_pmcs_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_subscriptions
    ADD CONSTRAINT user_pmcs_subscriptions_pkey PRIMARY KEY (subscriber_uid, checklist_id);


--
-- Name: user_pmcs_sync_state user_pmcs_sync_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_sync_state
    ADD CONSTRAINT user_pmcs_sync_state_pkey PRIMARY KEY (user_uid);


--
-- Name: user_suggestion_votes user_suggestion_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_suggestion_votes
    ADD CONSTRAINT user_suggestion_votes_pkey PRIMARY KEY (suggestion_id, voter_id);


--
-- Name: user_suggestions user_suggestions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_suggestions
    ADD CONSTRAINT user_suggestions_pkey PRIMARY KEY (id);


--
-- Name: shop_vehicle user_vehicle_copy1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle
    ADD CONSTRAINT user_vehicle_copy1_pkey PRIMARY KEY (id);


--
-- Name: shop_vehicle_notifications user_vehicle_notifications_copy1_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notifications
    ADD CONSTRAINT user_vehicle_notifications_copy1_pkey PRIMARY KEY (id);


--
-- Name: user_vehicle_notifications user_vehicle_notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_vehicle_notifications
    ADD CONSTRAINT user_vehicle_notifications_pkey PRIMARY KEY (id);


--
-- Name: user_vehicle user_vehicle_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_vehicle
    ADD CONSTRAINT user_vehicle_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (uid);


--
-- Name: amdf_phrase__idx_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX amdf_phrase__idx_niin ON public.amdf_phrase USING btree (niin);


--
-- Name: idx_amdf_i_and_s_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_amdf_i_and_s_niin ON public.amdf_i_and_s USING btree (niin);


--
-- Name: idx_amdf_management_lin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_amdf_management_lin ON public.amdf_management USING btree (lin);


--
-- Name: idx_amdf_management_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_amdf_management_niin ON public.amdf_management USING btree (niin);


--
-- Name: idx_analytics_event_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_analytics_event_count ON public.analytics_event_counters USING btree (event_type, count DESC);


--
-- Name: idx_analytics_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_analytics_event_type ON public.analytics_event_counters USING btree (event_type);


--
-- Name: idx_army_lin_to_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_army_lin_to_niin ON public.army_lin_to_niin USING btree (lin);


--
-- Name: idx_army_management_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_army_management_niin ON public.army_management USING btree (niin);


--
-- Name: idx_army_master_data_file_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_army_master_data_file_niin ON public.army_master_data_file USING btree (niin);


--
-- Name: idx_army_pack_suppl_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_army_pack_suppl_niin ON public.army_pack_supplemental_instruct USING btree (niin);


--
-- Name: idx_cancelled_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cancelled_niin ON public.flis_cancelled_niin USING btree (niin);


--
-- Name: idx_colloquial_name_inc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_colloquial_name_inc ON public.colloquial_name USING btree (inc);


--
-- Name: idx_component_end_item; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_component_end_item ON public.component_end_item USING btree (niin);


--
-- Name: idx_equipment_services_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_created_by ON public.equipment_services USING btree (created_by);


--
-- Name: idx_equipment_services_equipment_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_equipment_date ON public.equipment_services USING btree (equipment_id, service_date);


--
-- Name: idx_equipment_services_equipment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_equipment_id ON public.equipment_services USING btree (equipment_id);


--
-- Name: idx_equipment_services_is_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_is_completed ON public.equipment_services USING btree (is_completed);


--
-- Name: idx_equipment_services_service_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_service_date ON public.equipment_services USING btree (service_date);


--
-- Name: idx_equipment_services_service_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_service_type ON public.equipment_services USING btree (service_type);


--
-- Name: idx_equipment_services_shop_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_shop_completed ON public.equipment_services USING btree (shop_id, is_completed);


--
-- Name: idx_equipment_services_shop_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_shop_date ON public.equipment_services USING btree (shop_id, service_date);


--
-- Name: idx_equipment_services_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_equipment_services_shop_id ON public.equipment_services USING btree (shop_id);


--
-- Name: idx_faa_management_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_faa_management_niin ON public.faa_management USING btree (niin);


--
-- Name: idx_flis_item_characteristics_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_item_characteristics_niin ON public.flis_item_characteristics USING btree (niin);


--
-- Name: idx_flis_management; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_management ON public.flis_management USING btree (niin);


--
-- Name: idx_flis_packaging_1_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_packaging_1_niin ON public.flis_packaging_1 USING btree (niin);


--
-- Name: idx_flis_packaging_2_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_packaging_2_niin ON public.flis_packaging_2 USING btree (niin);


--
-- Name: idx_flis_phrase_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_phrase_niin ON public.flis_phrase USING btree (niin);


--
-- Name: idx_flis_reference_cage_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_reference_cage_code ON public.flis_reference USING btree (cage_code);


--
-- Name: idx_flis_reference_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_reference_niin ON public.flis_reference USING btree (niin);


--
-- Name: idx_flis_reference_part_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_flis_reference_part_number ON public.flis_reference USING btree (part_number);


--
-- Name: idx_help_code_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_help_code_trgm ON public.help USING gin (code public.gin_trgm_ops);


--
-- Name: idx_item_comments_comment_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_item_comments_comment_niin ON public.item_comments USING btree (comment_niin);


--
-- Name: idx_marine_corps_management; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_marine_corps_management ON public.marine_corps_management USING btree (niin);


--
-- Name: idx_marine_sl_62_item_id_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_marine_sl_62_item_id_niin ON public.marines_sl_6_2_item_id USING btree (niin);


--
-- Name: idx_marine_sl_62_item_supp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_marine_sl_62_item_supp ON public.marines_sl_6_2_item_supp USING btree (niin);


--
-- Name: idx_material_images_flagged; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_flagged ON public.material_images USING btree (is_flagged) WHERE (is_flagged = true);


--
-- Name: idx_material_images_flags_image; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_flags_image ON public.material_images_flags USING btree (image_id);


--
-- Name: idx_material_images_flags_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_flags_user ON public.material_images_flags USING btree (user_id);


--
-- Name: idx_material_images_net_votes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_net_votes ON public.material_images USING btree (net_votes DESC) WHERE (is_active = true);


--
-- Name: idx_material_images_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_niin ON public.material_images USING btree (niin) WHERE (is_active = true);


--
-- Name: idx_material_images_upload_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_upload_date ON public.material_images USING btree (upload_date DESC);


--
-- Name: idx_material_images_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_user_id ON public.material_images USING btree (user_id);


--
-- Name: idx_material_images_votes_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_votes_image_id ON public.material_images_votes USING btree (image_id);


--
-- Name: idx_material_images_votes_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_material_images_votes_user ON public.material_images_votes USING btree (user_id);


--
-- Name: idx_moe_rule; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_moe_rule ON public.moe_rule USING btree (niin);


--
-- Name: idx_notification_changes_field_changes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_changes_field_changes ON public.shop_vehicle_notification_changes USING gin (field_changes);


--
-- Name: idx_notification_changes_notification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_changes_notification_id ON public.shop_vehicle_notification_changes USING btree (notification_id);


--
-- Name: idx_notification_changes_shop_changes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_changes_shop_changes ON public.shop_vehicle_notification_changes USING btree (shop_id, changed_at DESC);


--
-- Name: idx_notification_changes_vehicle_changes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_changes_vehicle_changes ON public.shop_vehicle_notification_changes USING btree (vehicle_id, changed_at DESC);


--
-- Name: idx_notification_items_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_items_niin ON public.user_notification_items USING btree (niin);


--
-- Name: idx_notification_items_notification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_items_notification_id ON public.user_notification_items USING btree (notification_id);


--
-- Name: idx_notification_items_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_items_user_id ON public.user_notification_items USING btree (user_id);


--
-- Name: idx_nsn_cancelled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_nsn_cancelled ON public.nsn USING gin (cancelled_niin public.gin_trgm_ops);


--
-- Name: idx_part_number_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_part_number_niin ON public.part_number USING btree (niin);


--
-- Name: idx_part_number_partNumber; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "idx_part_number_partNumber" ON public.part_number USING btree (part_number);


--
-- Name: idx_ps_mag_summary_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ps_mag_summary_trgm ON public.ps_mag_summaries USING gin (summary public.gin_trgm_ops);


--
-- Name: idx_shop_invite_codes_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_invite_codes_active ON public.shop_invite_codes USING btree (is_active);


--
-- Name: idx_shop_invite_codes_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_invite_codes_code ON public.shop_invite_codes USING btree (code);


--
-- Name: idx_shop_invite_codes_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_invite_codes_created_by ON public.shop_invite_codes USING btree (created_by);


--
-- Name: idx_shop_invite_codes_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_invite_codes_shop_id ON public.shop_invite_codes USING btree (shop_id);


--
-- Name: idx_shop_list_items_added_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_list_items_added_by ON public.shop_list_items USING btree (added_by);


--
-- Name: idx_shop_list_items_list_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_list_items_list_id ON public.shop_list_items USING btree (list_id);


--
-- Name: idx_shop_list_items_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_list_items_niin ON public.shop_list_items USING btree (niin);


--
-- Name: idx_shop_lists_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_lists_created_by ON public.shop_lists USING btree (created_by);


--
-- Name: idx_shop_lists_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_lists_shop_id ON public.shop_lists USING btree (shop_id);


--
-- Name: idx_shop_members_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_role ON public.shop_members USING btree (role);


--
-- Name: idx_shop_members_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_shop_id ON public.shop_members USING btree (shop_id);


--
-- Name: idx_shop_members_shop_joined; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_shop_joined ON public.shop_members USING btree (shop_id, joined_at);


--
-- Name: idx_shop_members_shop_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_shop_user ON public.shop_members USING btree (shop_id, user_id);


--
-- Name: idx_shop_members_shop_user_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_shop_user_role ON public.shop_members USING btree (shop_id, user_id, role);


--
-- Name: idx_shop_members_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_members_user_id ON public.shop_members USING btree (user_id);


--
-- Name: idx_shop_messages_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_messages_created_at ON public.shop_messages USING btree (created_at);


--
-- Name: idx_shop_messages_shop_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_messages_shop_created ON public.shop_messages USING btree (shop_id, created_at DESC);


--
-- Name: idx_shop_messages_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_messages_shop_id ON public.shop_messages USING btree (shop_id);


--
-- Name: idx_shop_messages_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_messages_user_id ON public.shop_messages USING btree (user_id);


--
-- Name: idx_shop_notification_items_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_notification_items_niin ON public.shop_notification_items USING btree (niin);


--
-- Name: idx_shop_notification_items_notification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_notification_items_notification_id ON public.shop_notification_items USING btree (notification_id);


--
-- Name: idx_shop_notification_items_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_notification_items_shop_id ON public.shop_notification_items USING btree (shop_id);


--
-- Name: idx_shop_vehicle_creator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_creator_id ON public.shop_vehicle USING btree (creator_id);


--
-- Name: idx_shop_vehicle_notification_changes_changed_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notification_changes_changed_by ON public.shop_vehicle_notification_changes USING btree (changed_by);


--
-- Name: idx_shop_vehicle_notification_changes_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notification_changes_shop_id ON public.shop_vehicle_notification_changes USING btree (shop_id);


--
-- Name: idx_shop_vehicle_notifications_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notifications_completed ON public.shop_vehicle_notifications USING btree (completed);


--
-- Name: idx_shop_vehicle_notifications_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notifications_shop_id ON public.shop_vehicle_notifications USING btree (shop_id);


--
-- Name: idx_shop_vehicle_notifications_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notifications_type ON public.shop_vehicle_notifications USING btree (type);


--
-- Name: idx_shop_vehicle_notifications_vehicle_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_notifications_vehicle_id ON public.shop_vehicle_notifications USING btree (vehicle_id);


--
-- Name: idx_shop_vehicle_shop_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_shop_id ON public.shop_vehicle USING btree (shop_id);


--
-- Name: idx_shop_vehicle_shop_save_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shop_vehicle_shop_save_time ON public.shop_vehicle USING btree (shop_id, save_time DESC);


--
-- Name: idx_shops_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_shops_created_by ON public.shops USING btree (created_by);


--
-- Name: idx_standardization_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_standardization_niin ON public.flis_standardization USING btree (niin);


--
-- Name: idx_upload_limits_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upload_limits_time ON public.material_images_upload_limits USING btree (last_upload_time);


--
-- Name: idx_usable_on_codes_model; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usable_on_codes_model ON public.lookup_uoc USING btree (model);


--
-- Name: idx_usable_on_codes_uoc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usable_on_codes_uoc ON public.lookup_uoc USING btree (uoc);


--
-- Name: idx_user_items_categorized_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_items_categorized_user_id ON public.user_items_categorized USING btree (user_id);


--
-- Name: idx_user_items_serialized_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_items_serialized_user_id ON public.user_items_serialized USING btree (user_id);


--
-- Name: idx_user_suggestions_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_suggestions_created_at ON public.user_suggestions USING btree (created_at);


--
-- Name: idx_user_suggestions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_suggestions_user_id ON public.user_suggestions USING btree (user_id);


--
-- Name: idx_vehicle_admin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_admin ON public.user_vehicle USING btree (admin);


--
-- Name: idx_vehicle_notifications_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_notifications_completed ON public.user_vehicle_notifications USING btree (completed);


--
-- Name: idx_vehicle_notifications_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_notifications_type ON public.user_vehicle_notifications USING btree (type);


--
-- Name: idx_vehicle_notifications_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_notifications_user_id ON public.user_vehicle_notifications USING btree (user_id);


--
-- Name: idx_vehicle_notifications_vehicle_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_notifications_vehicle_id ON public.user_vehicle_notifications USING btree (vehicle_id);


--
-- Name: idx_vehicle_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehicle_user_id ON public.user_vehicle USING btree (user_id);


--
-- Name: item_comment_flags__idx_comment; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_comment_flags__idx_comment ON public.item_comment_flags USING btree (comment_id);


--
-- Name: item_comment_flags__idx_flagger; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_comment_flags__idx_flagger ON public.item_comment_flags USING btree (flagger_id);


--
-- Name: item_comments__idx_author; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_comments__idx_author ON public.item_comments USING btree (author_id);


--
-- Name: item_comments__idx_niin_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_comments__idx_niin_created ON public.item_comments USING btree (comment_niin, created_at);


--
-- Name: item_comments__idx_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX item_comments__idx_parent ON public.item_comments USING btree (parent_id);


--
-- Name: uq_analytics_event_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_analytics_event_key ON public.analytics_event_counters USING btree (event_type, entity_key);


--
-- Name: user_categories__idx_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_categories__idx_user_id ON public.user_item_category USING btree (user_uid);


--
-- Name: user_items_quick__index_niin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_items_quick__index_niin ON public.user_items_quick USING btree (niin);


--
-- Name: user_items_quick__index_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_items_quick__index_user_id ON public.user_items_quick USING btree (user_id);


--
-- Name: user_pmcs_checklists_owner_delta_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_checklists_owner_delta_idx ON public.user_pmcs_checklists USING btree (owner_uid, account_change_version);


--
-- Name: user_pmcs_community_releases_history_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_community_releases_history_idx ON public.user_pmcs_community_releases USING btree (checklist_id, released_at DESC);


--
-- Name: user_pmcs_community_sources_recent_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_community_sources_recent_idx ON public.user_pmcs_community_sources USING btree (updated_at DESC, checklist_id) WHERE (status = 'active'::text);


--
-- Name: user_pmcs_community_votes_voter_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_community_votes_voter_idx ON public.user_pmcs_community_votes USING btree (voter_uid);


--
-- Name: user_pmcs_inspection_comments_author_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_inspection_comments_author_id_idx ON public.user_pmcs_inspection_comments USING btree (author_id);


--
-- Name: user_pmcs_inspection_comments_pmcs_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_inspection_comments_pmcs_id_idx ON public.user_pmcs_inspection_comments USING btree (pmcs_id, created_at);


--
-- Name: user_pmcs_inspections_equipment_performed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_inspections_equipment_performed_idx ON public.user_pmcs_inspections USING btree (equipment_id, performed_date DESC);


--
-- Name: user_pmcs_inspections_performed_by_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_inspections_performed_by_idx ON public.user_pmcs_inspections USING btree (performed_by);


--
-- Name: user_pmcs_revision_models_lookup_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_revision_models_lookup_idx ON public.user_pmcs_revision_models USING btree (normalized_text, revision_id);


--
-- Name: user_pmcs_revision_models_search_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_revision_models_search_trgm_idx ON public.user_pmcs_revision_models USING gin (normalized_text public.gin_trgm_ops);


--
-- Name: user_pmcs_revisions_number_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_pmcs_revisions_number_idx ON public.user_pmcs_revisions USING btree (checklist_id, revision_number) WHERE (revision_number IS NOT NULL);


--
-- Name: user_pmcs_revisions_one_draft_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_pmcs_revisions_one_draft_idx ON public.user_pmcs_revisions USING btree (checklist_id) WHERE (state = 'draft'::text);


--
-- Name: user_pmcs_revisions_one_published_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_pmcs_revisions_one_published_idx ON public.user_pmcs_revisions USING btree (checklist_id) WHERE (state = 'published'::text);


--
-- Name: user_pmcs_subscriptions_active_pin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_subscriptions_active_pin_idx ON public.user_pmcs_subscriptions USING btree (installed_revision_id) WHERE (deleted_at IS NULL);


--
-- Name: user_pmcs_subscriptions_active_update_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_subscriptions_active_update_idx ON public.user_pmcs_subscriptions USING btree (subscriber_uid, checklist_id, installed_revision_id) WHERE (deleted_at IS NULL);


--
-- Name: user_pmcs_subscriptions_delta_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_subscriptions_delta_idx ON public.user_pmcs_subscriptions USING btree (subscriber_uid, account_change_version);


--
-- Name: user_pmcs_subscriptions_source_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_pmcs_subscriptions_source_idx ON public.user_pmcs_subscriptions USING btree (checklist_id, subscriber_uid);


--
-- Name: shop_vehicle_notification_changes fk_changed_by; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notification_changes
    ADD CONSTRAINT fk_changed_by FOREIGN KEY (changed_by) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: equipment_services fk_equipment_services_equipment; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equipment_services
    ADD CONSTRAINT fk_equipment_services_equipment FOREIGN KEY (equipment_id) REFERENCES public.shop_vehicle(id) ON DELETE CASCADE;


--
-- Name: equipment_services fk_equipment_services_shop; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equipment_services
    ADD CONSTRAINT fk_equipment_services_shop FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: equipment_services fk_equipment_services_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equipment_services
    ADD CONSTRAINT fk_equipment_services_user FOREIGN KEY (created_by) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: material_images_flags fk_image; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_flags
    ADD CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES public.material_images(id) ON DELETE CASCADE;


--
-- Name: material_images_votes fk_image; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_votes
    ADD CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES public.material_images(id) ON DELETE CASCADE;


--
-- Name: shop_vehicle_notification_changes fk_notification; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notification_changes
    ADD CONSTRAINT fk_notification FOREIGN KEY (notification_id) REFERENCES public.shop_vehicle_notifications(id) ON DELETE SET NULL;


--
-- Name: CONSTRAINT fk_notification ON shop_vehicle_notification_changes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT fk_notification ON public.shop_vehicle_notification_changes IS 'ON DELETE SET NULL preserves audit trail when notifications are deleted';


--
-- Name: shop_vehicle_notification_changes fk_shop; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notification_changes
    ADD CONSTRAINT fk_shop FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: material_images fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: material_images_flags fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_flags
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: material_images_upload_limits fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_upload_limits
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: material_images_votes fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.material_images_votes
    ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_pmcs_checklists fk_user_pmcs_checklists_owner; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_checklists
    ADD CONSTRAINT fk_user_pmcs_checklists_owner FOREIGN KEY (owner_uid) REFERENCES public.users(uid) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: user_pmcs_community_releases fk_user_pmcs_community_releases_revision; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_releases
    ADD CONSTRAINT fk_user_pmcs_community_releases_revision FOREIGN KEY (checklist_id, revision_id) REFERENCES public.user_pmcs_revisions(checklist_id, id) ON DELETE RESTRICT;


--
-- Name: user_pmcs_community_sources fk_user_pmcs_community_sources_checklist; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_sources
    ADD CONSTRAINT fk_user_pmcs_community_sources_checklist FOREIGN KEY (checklist_id) REFERENCES public.user_pmcs_checklists(id) ON DELETE RESTRICT;


--
-- Name: user_pmcs_community_sources fk_user_pmcs_community_sources_current_release; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_sources
    ADD CONSTRAINT fk_user_pmcs_community_sources_current_release FOREIGN KEY (checklist_id, current_release_revision_id) REFERENCES public.user_pmcs_community_releases(checklist_id, revision_id) ON DELETE RESTRICT;


--
-- Name: user_pmcs_community_votes fk_user_pmcs_community_votes_source; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_votes
    ADD CONSTRAINT fk_user_pmcs_community_votes_source FOREIGN KEY (checklist_id) REFERENCES public.user_pmcs_community_sources(checklist_id) ON DELETE CASCADE;


--
-- Name: user_pmcs_community_votes fk_user_pmcs_community_votes_voter; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_community_votes
    ADD CONSTRAINT fk_user_pmcs_community_votes_voter FOREIGN KEY (voter_uid) REFERENCES public.users(uid) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_pmcs_faults fk_user_pmcs_faults_pmcs_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_faults
    ADD CONSTRAINT fk_user_pmcs_faults_pmcs_id FOREIGN KEY (pmcs_id) REFERENCES public.user_pmcs_inspections(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_pmcs_inspection_comments fk_user_pmcs_inspection_comments_author_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspection_comments
    ADD CONSTRAINT fk_user_pmcs_inspection_comments_author_id FOREIGN KEY (author_id) REFERENCES public.users(uid) ON UPDATE CASCADE;


--
-- Name: user_pmcs_inspection_comments fk_user_pmcs_inspection_comments_pmcs_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspection_comments
    ADD CONSTRAINT fk_user_pmcs_inspection_comments_pmcs_id FOREIGN KEY (pmcs_id) REFERENCES public.user_pmcs_inspections(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_pmcs_inspections fk_user_pmcs_inspections_equipment_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspections
    ADD CONSTRAINT fk_user_pmcs_inspections_equipment_id FOREIGN KEY (equipment_id) REFERENCES public.shop_vehicle(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_pmcs_inspections fk_user_pmcs_inspections_performed_by; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_inspections
    ADD CONSTRAINT fk_user_pmcs_inspections_performed_by FOREIGN KEY (performed_by) REFERENCES public.users(uid) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: user_pmcs_items fk_user_pmcs_items_section; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_items
    ADD CONSTRAINT fk_user_pmcs_items_section FOREIGN KEY (section_id) REFERENCES public.user_pmcs_sections(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_notices fk_user_pmcs_notices_item; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_notices
    ADD CONSTRAINT fk_user_pmcs_notices_item FOREIGN KEY (item_id) REFERENCES public.user_pmcs_items(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_procedure_steps fk_user_pmcs_procedure_steps_item; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_procedure_steps
    ADD CONSTRAINT fk_user_pmcs_procedure_steps_item FOREIGN KEY (item_id) REFERENCES public.user_pmcs_items(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_revision_models fk_user_pmcs_revision_models_revision; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_revision_models
    ADD CONSTRAINT fk_user_pmcs_revision_models_revision FOREIGN KEY (revision_id) REFERENCES public.user_pmcs_revisions(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_revisions fk_user_pmcs_revisions_checklist; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_revisions
    ADD CONSTRAINT fk_user_pmcs_revisions_checklist FOREIGN KEY (checklist_id) REFERENCES public.user_pmcs_checklists(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_pmcs_section_models fk_user_pmcs_section_models_section; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_section_models
    ADD CONSTRAINT fk_user_pmcs_section_models_section FOREIGN KEY (section_id) REFERENCES public.user_pmcs_sections(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_sections fk_user_pmcs_sections_revision; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_sections
    ADD CONSTRAINT fk_user_pmcs_sections_revision FOREIGN KEY (revision_id) REFERENCES public.user_pmcs_revisions(id) ON DELETE CASCADE;


--
-- Name: user_pmcs_subscriptions fk_user_pmcs_subscriptions_installed_release; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_subscriptions
    ADD CONSTRAINT fk_user_pmcs_subscriptions_installed_release FOREIGN KEY (checklist_id, installed_revision_id) REFERENCES public.user_pmcs_community_releases(checklist_id, revision_id) ON DELETE RESTRICT;


--
-- Name: user_pmcs_subscriptions fk_user_pmcs_subscriptions_subscriber; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_subscriptions
    ADD CONSTRAINT fk_user_pmcs_subscriptions_subscriber FOREIGN KEY (subscriber_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_pmcs_sync_state fk_user_pmcs_sync_state_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pmcs_sync_state
    ADD CONSTRAINT fk_user_pmcs_sync_state_user FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shop_vehicle_notification_changes fk_vehicle; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notification_changes
    ADD CONSTRAINT fk_vehicle FOREIGN KEY (vehicle_id) REFERENCES public.shop_vehicle(id) ON DELETE SET NULL;


--
-- Name: CONSTRAINT fk_vehicle ON shop_vehicle_notification_changes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT fk_vehicle ON public.shop_vehicle_notification_changes IS 'ON DELETE SET NULL preserves audit trail when vehicles are deleted';


--
-- Name: item_comment_flags item_comment_flags_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comment_flags
    ADD CONSTRAINT item_comment_flags_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.item_comments(id);


--
-- Name: item_comment_flags item_comment_flags_flagger_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comment_flags
    ADD CONSTRAINT item_comment_flags_flagger_id_fkey FOREIGN KEY (flagger_id) REFERENCES public.users(uid);


--
-- Name: item_comments item_comments_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comments
    ADD CONSTRAINT item_comments_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(uid);


--
-- Name: item_comments item_comments_comment_niin_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comments
    ADD CONSTRAINT item_comments_comment_niin_fkey FOREIGN KEY (comment_niin) REFERENCES public.nsn(niin);


--
-- Name: item_comments item_comments_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item_comments
    ADD CONSTRAINT item_comments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.item_comments(id);


--
-- Name: shop_invite_codes shop_invite_codes_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_invite_codes
    ADD CONSTRAINT shop_invite_codes_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(uid);


--
-- Name: shop_invite_codes shop_invite_codes_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_invite_codes
    ADD CONSTRAINT shop_invite_codes_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: shop_list_items shop_list_items_added_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_list_items
    ADD CONSTRAINT shop_list_items_added_by_fkey FOREIGN KEY (added_by) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: shop_list_items shop_list_items_list_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_list_items
    ADD CONSTRAINT shop_list_items_list_id_fkey FOREIGN KEY (list_id) REFERENCES public.shop_lists(id) ON DELETE CASCADE;


--
-- Name: shop_lists shop_lists_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_lists
    ADD CONSTRAINT shop_lists_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: shop_lists shop_lists_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_lists
    ADD CONSTRAINT shop_lists_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: shop_members shop_members_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_members
    ADD CONSTRAINT shop_members_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: shop_members shop_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_members
    ADD CONSTRAINT shop_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: shop_messages shop_message_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_messages
    ADD CONSTRAINT shop_message_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.shop_messages(id) ON DELETE CASCADE;


--
-- Name: shop_messages shop_messages_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_messages
    ADD CONSTRAINT shop_messages_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON DELETE CASCADE;


--
-- Name: shop_messages shop_messages_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_messages
    ADD CONSTRAINT shop_messages_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: shop_notification_items shop_notification_items_notification_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_notification_items
    ADD CONSTRAINT shop_notification_items_notification_id_fkey FOREIGN KEY (notification_id) REFERENCES public.shop_vehicle_notifications(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shop_vehicle_notifications shop_vehicle_notifications_attached_list_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notifications
    ADD CONSTRAINT shop_vehicle_notifications_attached_list_fkey FOREIGN KEY (attached_shop_list) REFERENCES public.shop_lists(id) ON DELETE SET NULL;


--
-- Name: shop_vehicle_notifications shop_vehicle_notifications_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notifications
    ADD CONSTRAINT shop_vehicle_notifications_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shop_vehicle_notifications shop_vehicle_notifications_vehicle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle_notifications
    ADD CONSTRAINT shop_vehicle_notifications_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.shop_vehicle(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shops shops_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shops
    ADD CONSTRAINT shops_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(uid);


--
-- Name: user_item_category user_categories___fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_item_category
    ADD CONSTRAINT user_categories___fk_user_id FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_items_categorized user_items_category___fk_category; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_categorized
    ADD CONSTRAINT user_items_category___fk_category FOREIGN KEY (category_id, user_id) REFERENCES public.user_item_category(id, user_uid) ON DELETE CASCADE;


--
-- Name: user_items_categorized user_items_category___fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_categorized
    ADD CONSTRAINT user_items_category___fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_items_quick user_items_quick___fk_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_quick
    ADD CONSTRAINT user_items_quick___fk_user FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_items_serialized user_items_serialized___fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_items_serialized
    ADD CONSTRAINT user_items_serialized___fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_notification_items user_notification_items_notification_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_items
    ADD CONSTRAINT user_notification_items_notification_id_fkey FOREIGN KEY (notification_id) REFERENCES public.user_vehicle_notifications(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_suggestion_votes user_suggestion_votes_suggestion_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_suggestion_votes
    ADD CONSTRAINT user_suggestion_votes_suggestion_id_fkey FOREIGN KEY (suggestion_id) REFERENCES public.user_suggestions(id) ON DELETE CASCADE;


--
-- Name: shop_vehicle user_vehicle_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle
    ADD CONSTRAINT user_vehicle_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(uid) ON UPDATE SET NULL ON DELETE SET NULL;


--
-- Name: user_vehicle_notifications user_vehicle_notifications_vehicle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_vehicle_notifications
    ADD CONSTRAINT user_vehicle_notifications_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.user_vehicle(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shop_vehicle user_vehicle_shop_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_vehicle
    ADD CONSTRAINT user_vehicle_shop_id_fkey FOREIGN KEY (shop_id) REFERENCES public.shops(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_vehicle user_vehicle_user_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_vehicle
    ADD CONSTRAINT user_vehicle_user_id_fk FOREIGN KEY (user_id) REFERENCES public.users(uid) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

