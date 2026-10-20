package domain_test

import (
	"testing"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

func TestValidate_NormalizaElNombreATitleCase(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ya en title case, sin cambios", "Cuenta ARS", "Cuenta ARS"},
		{"todo en minuscula", "cuenta de mercadopago ars", "Cuenta De Mercadopago Ars"},
		{"palabra en mayuscula se preserva (acronimo/moneda)", "cuenta ARS", "Cuenta ARS"},
		{"espacios de mas se colapsan", "cuenta   de   mercadopago", "Cuenta De Mercadopago"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			acc := domain.Account{Name: tc.in, Type: domain.AccountTypeBank, Currency: domain.CurrencyARS}
			if err := acc.Validate(); err != nil {
				t.Fatalf("no se esperaba error: %v", err)
			}
			if acc.Name != tc.want {
				t.Fatalf("nombre normalizado = %q, se esperaba %q", acc.Name, tc.want)
			}
		})
	}
}

func TestValidate_NombreVacioSigueSiendoInvalido(t *testing.T) {
	acc := domain.Account{Name: "   ", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS}
	if err := acc.Validate(); err != domain.ErrAccountNameRequired {
		t.Fatalf("se esperaba ErrAccountNameRequired, se obtuvo: %v", err)
	}
}
