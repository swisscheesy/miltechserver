package mt

import (
	"time"
)

type AirForceManagement struct {
	ID     string  `bun:"niin,pk"`
	Fund   *string `bun:"fund"`
	Budget *string `bun:"budget"`
	Mmac   *string `bun:"mmac"`
	Pvc    *string `bun:"pvc"`
	Errc   *string `bun:"errc"`
}

type AmdfBilling struct {
	ID                       string  `bun:"niin,pk"`
	ServiceableCreditValue   *string `bun:"serviceable_credit_value"`
	UnserviceableCreditValue *string `bun:"unserviceable_credit_value"`
	ExchangePrice            *string `bun:"exchange_price"`
	ServiceableEpReturn      *string `bun:"serviceable_ep_return"`
	DeltaBill                *string `bun:"delta_bill"`
}

type AmdfCredit struct {
	ID        string  `bun:"niin,pk"`
	Aril      *string `bun:"aril"`
	ArilRic   *string `bun:"aril_ric"`
	DemilCode *string `bun:"demil_code"`
	AdpeCode  *string `bun:"adpe_code"`
	Pmic      *string `bun:"pmic"`
	Mr        *string `bun:"mr"`
	RecovCode *string `bun:"recov_code"`
	Esd       *string `bun:"esd"`
	Hmic      *string `bun:"hmic"`
	CritlCode *string `bun:"critl_code"`
}

type AmdfFreight struct {
	ID              string  `bun:"niin,pk"`
	ItemDescription *string `bun:"item_description"`
	Nmfc            *string `bun:"nmfc"`
	NmfcSub         *string `bun:"nmfc_sub"`
	Ltl             *string `bun:"ltl"`
	NmfDesc         *string `bun:"nmf_desc"`
	Stc             *string `bun:"stc"`
	Lcl             *string `bun:"lcl"`
	Rvc             *string `bun:"rvc"`
	Ufc             *string `bun:"ufc"`
	Adc             *string `bun:"adc"`
	Acc             *string `bun:"acc"`
	Wcc             *string `bun:"wcc"`
	Shc             *string `bun:"shc"`
	Tcc             *string `bun:"tcc"`
}

type AmdfIAnd struct {
	ID          int     `bun:"id,pk"`
	Niin        *string `bun:"niin"`
	Oou         *string `bun:"oou"`
	Jtc         *string `bun:"jtc"`
	RelatedFsc  *string `bun:"related_fsc"`
	RelatedNiin *string `bun:"related_niin"`
}

type AmdfManagement struct {
	ID     string  `bun:"niin,pk"`
	Scmc   *string `bun:"scmc"`
	Aec    *string `bun:"aec"`
	Matcat *string `bun:"matcat"`
	Lin    *string `bun:"lin"`
	Lcc    *string `bun:"lcc"`
	Ricc   *string `bun:"ricc"`
	Arc    *string `bun:"arc"`
	Src    *string `bun:"src"`
	Scic   *string `bun:"scic"`
	Ciic   *string `bun:"ciic"`
	Icc    *string `bun:"icc"`
	Slc    *string `bun:"slc"`
}

type AmdfMatcat struct {
	ID       string  `bun:"niin,pk"`
	Matcat1  *string `bun:"matcat_1"`
	Matcat2  *string `bun:"matcat_2"`
	Matcat3  *string `bun:"matcat_3"`
	Matcat45 *string `bun:"matcat_4_5"`
}

type AmdfPhrase struct {
	ID              int     `bun:"id,pk"`
	Niin            *string `bun:"niin"`
	PhraseCode      *string `bun:"phrase_code"`
	PhraseStatement *string `bun:"phrase_statement"`
	UiRel           *string `bun:"ui_rel"`
	UmRel           *string `bun:"um_rel"`
	QtyPerAssy      *string `bun:"qty_per_assy"`
}

type ArmyFreight struct {
	ID            string  `bun:"niin,pk"`
	UnNumber      *string `bun:"un_number"`
	MsdsIndicator *string `bun:"msds_indicator"`
}

type ArmyLinToNiin struct {
	Lin *string `bun:"lin"`
	ID  string  `bun:"niin,pk"`
}

type ArmyLineItemNumber struct {
	ID                 string  `bun:"lin,pk"`
	TypeAction         *string `bun:"type_action"`
	Cic                *string `bun:"cic"`
	Cmc                *string `bun:"cmc"`
	Ric                *string `bun:"ric"`
	PubDate            *string `bun:"pub_date"`
	Aps                *string `bun:"aps"`
	LinDeleteStatement *string `bun:"lin_delete_statement"`
	NewLin             *string `bun:"new_lin"`
	AndOr              *string `bun:"and_or"`
	ReplRatio          *string `bun:"repl_ratio"`
	TypeClass          *string `bun:"type_class"`
	AddChgDelDate      *string `bun:"add_chg_del_date"`
	NsnDeleteStatement *string `bun:"nsn_delete_statement"`
	AssignedNiin       *string `bun:"assigned_niin"`
}

type ArmyManagement struct {
	ID       int     `bun:"id,pk"`
	Niin     *string `bun:"niin"`
	Matcat1  *string `bun:"matcat_1"`
	Matcat2  *string `bun:"matcat_2"`
	Matcat3  *string `bun:"matcat_3"`
	Matcat45 *string `bun:"matcat_4_5"`
	Arc      *string `bun:"arc"`
}

type ArmyMasterDatumFile struct {
	ID            string  `bun:"niin,pk"`
	Fsc           *string `bun:"fsc"`
	Nomenclature  *string `bun:"nomenclature"`
	Act           *string `bun:"act"`
	Addl          *string `bun:"addl"`
	Sos           *string `bun:"sos"`
	Aac           *string `bun:"aac"`
	Psc           *string `bun:"psc"`
	ArmyUnitPrice *string `bun:"army_unit_price"`
	Ui            *string `bun:"ui"`
	Fc            *string `bun:"fc"`
	Um            *string `bun:"um"`
	MeasQty       *string `bun:"meas_qty"`
	Eic           *string `bun:"eic"`
	Ec            *string `bun:"ec"`
}

type ArmyPackSupplementalInstruct struct {
	ID                       int     `bun:"id,pk"`
	Niin                     *string `bun:"niin"`
	SupplementalInstructions *string `bun:"supplemental_instructions"`
}

type ArmyPackaging1 struct {
	ID         string  `bun:"niin,pk"`
	Mop        *string `bun:"mop"`
	ClngDrying *string `bun:"clng_drying"`
	PresMat    *string `bun:"pres_mat"`
	WrapMat    *string `bun:"wrap_mat"`
	CushDun    *string `bun:"cush_dun"`
	Thk        *string `bun:"thk"`
	UnitCont   *string `bun:"unit_cont"`
	InterCont  *string `bun:"inter_cont"`
	Opi        *string `bun:"opi"`
	SpcMkg     *string `bun:"spc_mkg"`
	Ucl        *string `bun:"ucl"`
	LvlA       *string `bun:"lvl_a"`
	LvlB       *string `bun:"lvl_b"`
	LvlC       *string `bun:"lvl_c"`
}

type ArmyPackaging2 struct {
	ID              string  `bun:"niin,pk"`
	PkgCat          *string `bun:"pkg_cat"`
	UnpkgItemWeight *string `bun:"unpkg_item_weight"`
	UnpkgItemDim    *string `bun:"unpkg_item_dim"`
	DrwgPn          *string `bun:"drwg_pn"`
	CageCode        *string `bun:"cage_code"`
}

type ArmyPackagingAndFreight struct {
	ID             string  `bun:"niin,pk"`
	Lop            *string `bun:"lop"`
	PkgRef         *string `bun:"pkg_ref"`
	Upq            *string `bun:"upq"`
	Icq            *string `bun:"icq"`
	Tos            *string `bun:"tos"`
	Haz            *string `bun:"haz"`
	UnitPackSize   *string `bun:"unit_pack_size"`
	UnitPackWeight *string `bun:"unit_pack_weight"`
	UnitPackCube   *string `bun:"unit_pack_cube"`
	PkgInd         *string `bun:"pkg_ind"`
	PkLvlRefInd    *string `bun:"pk_lvl_ref_ind"`
}

type ArmyPackagingSpecialInstruct struct {
	ID            string  `bun:"niin,pk"`
	ContNsn       *string `bun:"cont_nsn"`
	SpiNo         *string `bun:"spi_no"`
	SpiRev        *string `bun:"spi_rev"`
	SpiDate       *string `bun:"spi_date"`
	PkgDesignActy *string `bun:"pkg_design_acty"`
}

type ArmyRelatedNsn struct {
	ID                 string  `bun:"niin,pk"`
	Lin                *string `bun:"lin"`
	Nomenclature       *string `bun:"nomenclature"`
	RelatedNiin        *string `bun:"related_niin"`
	Tfc                *string `bun:"tfc"`
	ArmyTypeDesignator *string `bun:"army_type_designator"`
	TypeClass          *string `bun:"type_class"`
	Mscr               *string `bun:"mscr"`
	ReferenceData      *string `bun:"reference_data"`
}

type ArmySarsscat struct {
	ID           string  `bun:"niin,pk"`
	InCode       *string `bun:"in_code"`
	Muc          *string `bun:"muc"`
	Wrty         *string `bun:"wrty"`
	UiOld        *string `bun:"ui_old"`
	UiConvFactor *string `bun:"ui_conv_factor"`
	Aimi         *string `bun:"aimi"`
	Lop          *string `bun:"lop"`
	SpStrg       *string `bun:"sp_strg"`
	Temp         *string `bun:"temp"`
	Slc          *string `bun:"slc"`
	RelatedNiin  *string `bun:"related_niin"`
	ArilRic1     *string `bun:"aril_ric_1"`
	ArilRic2     *string `bun:"aril_ric_2"`
	ArilRic3     *string `bun:"aril_ric_3"`
	ArilRic4     *string `bun:"aril_ric_4"`
	ArilRic5     *string `bun:"aril_ric_5"`
}

type ArmySubstituteLin struct {
	ID            int     `bun:"id,pk"`
	Lin           *string `bun:"lin"`
	SubstituteLin *string `bun:"substitute_lin"`
}

type CageAddress struct {
	ID             string  `bun:"cage_code,pk"`
	CompanyName    *string `bun:"company_name"`
	CompanyName2   *string `bun:"company_name_2"`
	CompanyName3   *string `bun:"company_name_3"`
	CompanyName4   *string `bun:"company_name_4"`
	CompanyName5   *string `bun:"company_name_5"`
	StreetAddress1 *string `bun:"street_address_1"`
	StreetAddress2 *string `bun:"street_address_2"`
	PoBox          *string `bun:"po_box"`
	City           *string `bun:"city"`
	State          *string `bun:"state"`
	Zip            *string `bun:"zip"`
	Country        *string `bun:"country"`
	DateEst        *string `bun:"date_est"`
	LastUpdate     *string `bun:"last_update"`
	FormerName1    *string `bun:"former_name_1"`
	FormerName2    *string `bun:"former_name_2"`
	FormerName3    *string `bun:"former_name_3"`
	FormerName4    *string `bun:"former_name_4"`
	FrnDom         *string `bun:"frn_dom"`
}

type CageStatusAndType struct {
	ID              string  `bun:"cage_code,pk"`
	Status          *string `bun:"status"`
	Type            *string `bun:"type"`
	Cao             *string `bun:"cao"`
	Adp             *string `bun:"adp"`
	Phone           *string `bun:"phone"`
	Fax             *string `bun:"fax"`
	RplmCode        *string `bun:"rplm_code"`
	AssocCode       *string `bun:"assoc_code"`
	AffilCode       *string `bun:"affil_code"`
	BusSize         *string `bun:"bus_size"`
	PrimaryBusiness *string `bun:"primary_business"`
	TypeOfBusiness  *string `bun:"type_of_business"`
	WomanOwned      *string `bun:"woman_owned"`
	CngrslDstrct    *string `bun:"cngrsl_dstrct"`
	Designator      *string `bun:"designator"`
}

type CoastGuardManagement struct {
	ID   string  `bun:"niin,pk"`
	Iac  *string `bun:"iac"`
	Snc  *string `bun:"snc"`
	Smcc *string `bun:"smcc"`
}

type ColloquialName struct {
	ID             int     `bun:"id,pk"`
	Inc            *string `bun:"inc"`
	RelatedInc     *string `bun:"related_inc"`
	ColloquialName *string `bun:"colloquial_name"`
}

type ComponentEndItem struct {
	ID           int     `bun:"id,pk"`
	Niin         *string `bun:"niin"`
	WpnSysID     *string `bun:"wpn_sys_id"`
	WpnSysSvc    *string `bun:"wpn_sys_svc"`
	WpnSysInd    *string `bun:"wpn_sys_ind"`
	WpnSysEsntl  *string `bun:"wpn_sys_esntl"`
	WeaponSystem *string `bun:"weapon_system"`
}

type Disposition struct {
	ID            string  `bun:"niin,pk"`
	DemilCode     *string `bun:"demil_code"`
	DemilIntg     *string `bun:"demil_intg"`
	FscFlagType   *string `bun:"fsc_flag_type"`
	NiinFlagType  *string `bun:"niin_flag_type"`
	OtherFlagType *string `bun:"other_flag_type"`
	DoNotSell     *string `bun:"do_not_sell"`
	SafeToSell    *string `bun:"safe_to_sell"`
}

type DssWeightAndCube struct {
	ID        string  `bun:"niin,pk"`
	DssWeight *string `bun:"dss_weight"`
	DssCube   *string `bun:"dss_cube"`
}

type FaaManagement struct {
	ID          int     `bun:"id,pk"`
	Niin        *string `bun:"niin"`
	RetailPrice *string `bun:"retail_price"`
	DodPrice    *string `bun:"dod_price"`
	RepairPrice *string `bun:"repair_price"`
	Moe         *string `bun:"moe"`
}

type FliCancelledNiin struct {
	ID               int     `bun:"id,pk"`
	Niin             *string `bun:"niin"`
	CancelledNiinFsc *string `bun:"cancelled_niin_fsc"`
	CancelledNiin    *string `bun:"cancelled_niin"`
	NiinStatCd       *string `bun:"niin_stat_cd"`
	EffDate          *string `bun:"eff_date"`
	Demil            *string `bun:"demil"`
}

type FliFreight struct {
	ID      string  `bun:"niin,pk"`
	ActyCd  *string `bun:"acty_cd"`
	Integ   *string `bun:"integ"`
	Nmfc    *string `bun:"nmfc"`
	NmfcSub *string `bun:"nmfc_sub"`
	Ufc     *string `bun:"ufc"`
	Rvc     *string `bun:"rvc"`
	Hmc     *string `bun:"hmc"`
	Ltl     *string `bun:"ltl"`
	Lcl     *string `bun:"lcl"`
	Wcc     *string `bun:"wcc"`
	Tcc     *string `bun:"tcc"`
	Shc     *string `bun:"shc"`
	Adc     *string `bun:"adc"`
	Acc     *string `bun:"acc"`
	Ash     *string `bun:"ash"`
	NmfDesc *string `bun:"nmf_desc"`
}

type FliIdentification struct {
	ID           string  `bun:"niin,pk"`
	TypeIi       *string `bun:"type_ii"`
	Inc          *string `bun:"inc"`
	Hcc          *string `bun:"hcc"`
	Isc          *string `bun:"isc"`
	StandardNiin *string `bun:"standard_niin"`
}

type FlisItemCharacteristic struct {
	ID                    int     `bun:"id,pk"`
	Niin                  *string `bun:"niin"`
	Mrc                   *string `bun:"mrc"`
	RequirementsStatement *string `bun:"requirements_statement"`
	ClearTextReply        *string `bun:"clear_text_reply"`
}

type FliManagement struct {
	ID            int     `bun:"id,pk"`
	Niin          *string `bun:"niin"`
	EffectiveDate *string `bun:"effective_date"`
	Moe           *string `bun:"moe"`
	Sos           *string `bun:"sos"`
	Sosm          *string `bun:"sosm"`
	Aac           *string `bun:"aac"`
	Qup           *string `bun:"qup"`
	Ui            *string `bun:"ui"`
	UiConvFac     *string `bun:"ui_conv_fac"`
	UnitPrice     *string `bun:"unit_price"`
	Slc           *string `bun:"slc"`
	Ciic          *string `bun:"ciic"`
	RecRepCode    *string `bun:"rec_rep_code"`
	MgmtCtl       *string `bun:"mgmt_ctl"`
	RepNetPr      *string `bun:"rep_net_pr"`
	Usc           *string `bun:"usc"`
}

type FliManagementId struct {
	ID            string  `bun:"niin,pk"`
	Fiig          *string `bun:"fiig"`
	Pmic          *string `bun:"pmic"`
	AdpeCode      *string `bun:"adpe_code"`
	CritlCode     *string `bun:"critl_code"`
	RpdMrc        *string `bun:"rpd_mrc"`
	DemilCode     *string `bun:"demil_code"`
	DemilIntg     *string `bun:"demil_intg"`
	NiinAsgmt     *string `bun:"niin_asgmt"`
	EstAct        *string `bun:"est_act"`
	EstActDate    *string `bun:"est_act_date"`
	Esd           *string `bun:"esd"`
	Hmic          *string `bun:"hmic"`
	Enac          *string `bun:"enac"`
	ScheduleB     *string `bun:"schedule_b"`
	Inc           *string `bun:"inc"`
	Pinc          *string `bun:"pinc"`
	MinRlseQty    *string `bun:"min_rlse_qty"`
	Sla           *string `bun:"sla"`
	UiConvFactor  *string `bun:"ui_conv_factor"`
	Fedmall       *string `bun:"fedmall"`
	IuidIndicator *string `bun:"iuid_indicator"`
	LstKwnSos     *string `bun:"lst_kwn_sos"`
	NatoFmsn      *string `bun:"nato_fmsn"`
}

type FliPackaging1 struct {
	ID            int     `bun:"id,pk"`
	Niin          *string `bun:"niin"`
	PicaSica      *string `bun:"pica_sica"`
	Ui            *string `bun:"ui"`
	Tos           *string `bun:"tos"`
	Icq           *string `bun:"icq"`
	Mop           *string `bun:"mop"`
	ClngDrying    *string `bun:"clng_drying"`
	PresMat       *string `bun:"pres_mat"`
	WrapMat       *string `bun:"wrap_mat"`
	PkgCat        *string `bun:"pkg_cat"`
	PkgDesignActy *string `bun:"pkg_design_acty"`
	CushDun       *string `bun:"cush_dun"`
	Thk           *string `bun:"thk"`
	UnitCont      *string `bun:"unit_cont"`
	PkgDataSource *string `bun:"pkg_data_source"`
	InterCont     *string `bun:"inter_cont"`
	Ucl           *string `bun:"ucl"`
}

type FliPackaging2 struct {
	ID                       int     `bun:"id,pk"`
	Niin                     *string `bun:"niin"`
	PicaSica                 *string `bun:"pica_sica"`
	SpcMkg                   *string `bun:"spc_mkg"`
	LvlA                     *string `bun:"lvl_a"`
	LvlB                     *string `bun:"lvl_b"`
	LvlC                     *string `bun:"lvl_c"`
	UnitPackWeight           *string `bun:"unit_pack_weight"`
	UnitPackSize             *string `bun:"unit_pack_size"`
	UnitPackCube             *string `bun:"unit_pack_cube"`
	ContNsn                  *string `bun:"cont_nsn"`
	Opi                      *string `bun:"opi"`
	UnpkgItemDim             *string `bun:"unpkg_item_dim"`
	UnpkgItemWeight          *string `bun:"unpkg_item_weight"`
	SpiDate                  *string `bun:"spi_date"`
	SpiNo                    *string `bun:"spi_no"`
	SpiRev                   *string `bun:"spi_rev"`
	SupplementalInstructions *string `bun:"supplemental_instructions"`
}

type FliPhrase struct {
	ID              int     `bun:"id,pk"`
	Niin            *string `bun:"niin"`
	Moe             *string `bun:"moe"`
	Usc             *string `bun:"usc"`
	PhraseCode      *string `bun:"phrase_code"`
	PhraseStatement *string `bun:"phrase_statement"`
	Oou             *string `bun:"oou"`
	Jtc             *string `bun:"jtc"`
	Qpa             *string `bun:"qpa"`
	Um              *string `bun:"um"`
	TechDocNbr      *string `bun:"tech_doc_nbr"`
	QntvExprsn      *string `bun:"qntv_exprsn"`

	NiinRel *Nsn `bun:"join:niin=niin,rel:belongs-to"`
}

type FliReference struct {
	ID         int     `bun:"id,pk"`
	Niin       *string `bun:"niin"`
	PartNumber *string `bun:"part_number"`
	CageCode   *string `bun:"cage_code"`
	Status     *string `bun:"status"`
	Rncc       *string `bun:"rncc"`
	Rnvc       *string `bun:"rnvc"`
	Dac        *string `bun:"dac"`
	Rnaac      *string `bun:"rnaac"`
	Rnfc       *string `bun:"rnfc"`
	Rnsc       *string `bun:"rnsc"`
	Rnjc       *string `bun:"rnjc"`
	Msds       *string `bun:"msds"`
	Sadc       *string `bun:"sadc"`
	Medals     *string `bun:"medals"`
}

type FliStandardization struct {
	ID           int     `bun:"id,pk"`
	Niin         *string `bun:"niin"`
	RelatedNsn   *string `bun:"related_nsn"`
	Isc          *string `bun:"isc"`
	OrigStdznDec *string `bun:"orig_stdzn_dec"`
	DtStdznDec   *string `bun:"dt_stdzn_dec"`
	NiinStatCd   *string `bun:"niin_stat_cd"`
}

type LookupUoc struct {
	Uoc   string `bun:"uoc,pk"`
	Model string `bun:"model,pk"`
}

type MarineCorpManagement struct {
	ID   int     `bun:"id,pk"`
	Niin *string `bun:"niin"`
	Sac  *string `bun:"sac"`
	Cec  *string `bun:"cec"`
	Mec  *string `bun:"mec"`
	Mic  *string `bun:"mic"`
	Otc  *string `bun:"otc"`
	Pcc  *string `bun:"pcc"`
}

type MarineMhif struct {
	ID          string  `bun:"niin,pk"`
	Ui          *string `bun:"ui"`
	Sac         *string `bun:"sac"`
	Mec         *string `bun:"mec"`
	Slc         *string `bun:"slc"`
	McRecCode   *string `bun:"mc_rec_code"`
	Ciic        *string `bun:"ciic"`
	PhraseCode  *string `bun:"phrase_code"`
	Pmic        *string `bun:"pmic"`
	McCic       *string `bun:"mc_cic"`
	Pcc         *string `bun:"pcc"`
	Cec         *string `bun:"cec"`
	Sub         *string `bun:"sub"`
	DemilCode   *string `bun:"demil_code"`
	Sos         *string `bun:"sos"`
	AdpeCode    *string `bun:"adpe_code"`
	Aac         *string `bun:"aac"`
	Mic         *string `bun:"mic"`
	MhifDate    *string `bun:"mhif_date"`
	McItemName  *string `bun:"mc_item_name"`
	PrimeFsc    *string `bun:"prime_fsc"`
	PrimeNiin   *string `bun:"prime_niin"`
	McUnitPrice *string `bun:"mc_unit_price"`
}

type MarineSl61 struct {
	ID              string  `bun:"niin,pk"`
	Idn             *string `bun:"idn"`
	TypeModelNumber *string `bun:"type_model_number"`
	TamNumber       *string `bun:"tam_number"`
	InServiceDate   *string `bun:"in_service_date"`
	ActSch          *string `bun:"act_sch"`
	ExitDate        *string `bun:"exit_date"`
	Spc             *string `bun:"spc"`
	Tidc            *string `bun:"tidc"`
	Wsc             *string `bun:"wsc"`
	Cec             *string `bun:"cec"`
	Lap             *string `bun:"lap"`
	Alo             *string `bun:"alo"`
	McNomenclature  *string `bun:"mc_nomenclature"`
	RepairPartCount *string `bun:"repair_part_count"`
}

type MarineSl62ItemId struct {
	ID               int     `bun:"id,pk"`
	Niin             *string `bun:"niin"`
	Idn              *string `bun:"idn"`
	ApprovedItemName *string `bun:"approved_item_name"`
	Smr              *string `bun:"smr"`
	ExitDate         *string `bun:"exit_date"`
	Cec              *string `bun:"cec"`
}

type MarineSl62ItemSupp struct {
	ID   int     `bun:"id,pk"`
	Niin *string `bun:"niin"`
	Idn  *string `bun:"idn"`
	Qty3 *string `bun:"qty3"`
	Qty4 *string `bun:"qty4"`
	Mc   *string `bun:"mc"`
	Um   *string `bun:"um"`
	Ptrf *string `bun:"ptrf"`
	Wsc  *string `bun:"wsc"`
	Crit *string `bun:"crit"`
}

type MarineStockList63 struct {
	ID         string  `bun:"niin,pk"`
	Ssr        *string `bun:"ssr"`
	Cm         *string `bun:"cm"`
	Uur        *string `bun:"uur"`
	Cei        *string `bun:"cei"`
	Bii        *string `bun:"bii"`
	Aal        *string `bun:"aal"`
	Cli        *string `bun:"cli"`
	Sl3Remarks *string `bun:"sl3_remarks"`
}

type MoeRule struct {
	ID         int     `bun:"id,pk"`
	Niin       *string `bun:"niin"`
	MoeRl      *string `bun:"moe_rl"`
	MoeCd      *string `bun:"moe_cd"`
	Amc        *string `bun:"amc"`
	Amsc       *string `bun:"amsc"`
	Nimsc      *string `bun:"nimsc"`
	DtAsgnd    *string `bun:"dt_asgnd"`
	Imc        *string `bun:"imc"`
	Imca       *string `bun:"imca"`
	Aac        *string `bun:"aac"`
	Pica       *string `bun:"pica"`
	PicaLoa    *string `bun:"pica_loa"`
	Sica       *string `bun:"sica"`
	SicaLoa    *string `bun:"sica_loa"`
	Submtr     *string `bun:"submtr"`
	AuthCollab *string `bun:"auth_collab"`
	SuppCollab *string `bun:"supp_collab"`
	AuthRcvr   *string `bun:"auth_rcvr"`
	SuppRcvr   *string `bun:"supp_rcvr"`
	Dsor       *string `bun:"dsor"`
	FmrMoeRl   *string `bun:"fmr_moe_rl"`
}

type NavyManagement struct {
	ID   string  `bun:"niin,pk"`
	Cog  *string `bun:"cog"`
	Smic *string `bun:"smic"`
	Irrc *string `bun:"irrc"`
	Smcc *string `bun:"smcc"`
}

type Nsn struct {
	Service       string  `bun:"service,nullzero"`
	Category      string  `bun:"category,nullzero"`
	Fsc           string  `bun:"fsc,nullzero"`
	ID            string  `bun:"niin,pk"`
	CancelledNiin *string `bun:"cancelled_niin"`
	ItemName      *string `bun:"item_name"`
	Inc           *string `bun:"inc"`
}

type PartNumber struct {
	ID              int     `bun:"id,pk"`
	Niin            *string `bun:"niin"`
	Fsc             *string `bun:"fsc"`
	ItemName        *string `bun:"item_name"`
	CageCode        *string `bun:"cage_code"`
	CompanyName     *string `bun:"company_name"`
	PartNumber      *string `bun:"part_number"`
	PublicationDate *string `bun:"publication_date"`
}

type QuickListBattery struct {
	ID          string  `bun:"nsn,pk"`
	PartNumber  *string `bun:"part_number"`
	Model       *string `bun:"model"`
	Description *string `bun:"description"`
}

type QuickListClothing struct {
	ID          string  `bun:"nsn,pk"`
	Size        *string `bun:"size"`
	Description *string `bun:"description"`
}

type QuickListWheelTire struct {
	ID          int     `bun:"id,pk"`
	Vehicle     *string `bun:"vehicle"`
	AssemblyNsn *string `bun:"assembly_nsn"`
	TireNsn     *string `bun:"tire_nsn"`
	Size        *string `bun:"size"`
	ItemComment *string `bun:"item_comment"`
}

type SocomManagement struct {
	ID    string  `bun:"niin,pk"`
	Icp   *string `bun:"icp"`
	AL    *string `bun:"a_l"`
	Rep   *string `bun:"rep"`
	WEsdc *string `bun:"w_esdc"`
	Ar    *string `bun:"ar"`
	Cos   *string `bun:"cos"`
}

type UserItemCategory struct {
	Uuid          string  `bun:"uuid,pk"`
	UserUid       string  `bun:"user_uid,pk"`
	Name          string  `bun:"name,nullzero"`
	Comment       *string `bun:"comment"`
	ImageLocation *string `bun:"image_location"`

	UserUidRel *User `bun:"join:user_uid=uid,rel:belongs-to"`
}

type UserItemComment struct {
	ID          int        `bun:"id,pk"`
	AuthorID    string     `bun:"author_id,nullzero"`
	Text        *string    `bun:"text"`
	ParentID    *int       `bun:"parent_id"`
	CommentNiin *string    `bun:"comment_niin"`
	CreatedAt   *time.Time `bun:"created_at"`

	Author *User            `bun:"join:author_id=uid,rel:belongs-to"`
	Parent *UserItemComment `bun:"join:parent_id=id,rel:belongs-to"`
}

type UserItemCategorized struct {
	UserID        *string    `bun:"user_id"`
	Niin          string     `bun:"niin,pk"`
	ItemName      *string    `bun:"item_name"`
	Quantity      *int       `bun:"quantity"`
	EquipModel    *string    `bun:"equip_model"`
	Uoc           *string    `bun:"uoc"`
	CategoryID    string     `bun:"category_id,pk"`
	SaveTime      *time.Time `bun:"save_time"`
	ImageLocation *string    `bun:"image_location"`

	User         *User             `bun:"join:user_id=uid,rel:belongs-to"`
	NiinRel      *Nsn              `bun:"join:niin=niin,rel:belongs-to"`
	UserCategory *UserItemCategory `bun:"join:category_id=uuid,rel:belongs-to,-"` // unsupported
}

type UserItemQuick struct {
	UserID        string     `bun:"user_id,pk"`
	Niin          string     `bun:"niin,pk"`
	ItemName      *string    `bun:"item_name"`
	ImageLocation *string    `bun:"image_location"`
	ItemComment   *string    `bun:"item_comment"`
	SaveTime      *time.Time `bun:"save_time"`

	User    *User `bun:"join:user_id=uid,rel:belongs-to"`
	NiinRel *Nsn  `bun:"join:niin=niin,rel:belongs-to"`
}

type UserItemSerialized struct {
	UserID        string     `bun:"user_id,pk"`
	Niin          string     `bun:"niin,pk"`
	ItemName      *string    `bun:"item_name"`
	Serial        string     `bun:"serial,pk"`
	ImageLocation *string    `bun:"image_location"`
	SaveTime      *time.Time `bun:"save_time"`
	ItemComment   *string    `bun:"item_comment"`

	User    *User `bun:"join:user_id=uid,rel:belongs-to"`
	NiinRel *Nsn  `bun:"join:niin=niin,rel:belongs-to"`
}

type User struct {
	ID        string     `bun:"uid,pk"`
	Email     string     `bun:"email,nullzero"`
	Username  string     `bun:"username,nullzero"`
	CreatedAt *time.Time `bun:"created_at"`
	IsEnabled *bool      `bun:"is_enabled"`
	LastLogin *time.Time `bun:"last_login"`
}
