package irs

func (irs *IRS) Delta(tenor float64) float64{
	irs.CCPRates[tenor] -= 0.01
	npv1 := irs.NPV(BuildCurve(irs.SettlementDate, irs.CCPRates))

	irs.CCPRates[tenor] += 0.02
	npv2 := irs.NPV(BuildCurve(irs.SettlementDate, irs.CCPRates))

	irs.CCPRates[tenor] -= 0.01

	return (npv1-npv2)/2
}