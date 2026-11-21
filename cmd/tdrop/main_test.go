package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionImprimeVersaoEProtocolo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 {
		t.Fatalf("run(version) = %d, esperado 0 (stderr: %s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "protocolo 1") {
		t.Fatalf("saída = %q, esperado versão do protocolo", out.String())
	}
}

func TestSemArgumentosRetornaUsoComCodigoDeErro(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("run() = %d, esperado 2", code)
	}
	if !strings.Contains(errb.String(), "uso:") {
		t.Fatalf("stderr = %q, esperado mensagem de uso", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout deveria estar vazio em erro, obteve %q", out.String())
	}
}

func TestComandoDesconhecidoRetornaCodigoDeErro(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"inexistente"}, &out, &errb); code != 2 {
		t.Fatalf("run(inexistente) = %d, esperado 2", code)
	}
}
