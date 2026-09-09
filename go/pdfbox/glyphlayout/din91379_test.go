package glyphlayout_test

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// GlyphLayoutDin91379Test, which is the widest Latin case the Java has: every
// letter DIN 91379 names for a name written in Europe, and then the sequences
// -- a letter and one or two combining marks over it, which is mark
// positioning on a script where nothing else in these tests exercises it.
//
// The Java asserts the page count and renders the page against the reference
// PDF; the extracted text it writes to a file with a TODO saying the comparison
// is "Not yet correct as of 4.7.2026", so there is nothing there to port.

// latinCharsDin91379 is GlyphLayoutDin91379Test.LATIN_CHARS_DIN_91379, taken
// from the Java source line for line.
var latinCharsDin91379 = strings.Join([]string{
	"DIN 91379: Characters in Unicode for the electronic processing of names\n",
	"and data exchange in Europe\n",
	"Font used: Arimo-Regular.ttf\n",
	"bll; Latin Letters (normative)\n",
	"A B C D E F G H I J K L M N O P Q R S T U V W X Y Z a b c d e f g h i j k l m n o p q r s t u v w x y z\n",
	"À Á Â Ã Ä Å Æ Ç È É Ê Ë Ì Í Î Ï Ð Ñ Ò Ó Ô Õ Ö Ø Ù Ú Û Ü Ý Þ ß à á â ã ä å æ ç è é ê ë ì í î ï ð ñ ò ó\n",
	"ô õ ö ø ù ú û ü ý þ ÿ Ā ā Ă ă Ą ą Ć ć Ĉ ĉ Ċ ċ Č č Ď ď Đ đ Ē ē Ĕ ĕ Ė ė Ę ę Ě ě Ĝ ĝ Ğ ğ Ġ ġ Ģ ģ Ĥ ĥ Ħ\n",
	"ħ Ĩ ĩ Ī ī Ĭ ĭ Į į İ ı Ĳ ĳ Ĵ ĵ Ķ ķ ĸ Ĺ ĺ Ļ ļ Ľ ľ Ŀ ŀ Ł ł Ń ń Ņ ņ Ň ň ŉ Ŋ ŋ Ō ō Ŏ ŏ Ő ő Œ œ Ŕ ŕ Ŗ ŗ Ř ř Ś ś Ŝ ŝ Ş\n",
	"ş Š š Ţ ţ Ť ť Ŧ ŧ Ũ ũ Ū ū Ŭ ŭ Ů ů Ű ű Ų ų Ŵ ŵ Ŷ ŷ Ÿ Ź ź Ż ż Ž ž Ƈ ƈ Ə Ɨ Ơ ơ Ư ư Ʒ Ǎ ǎ Ǐ ǐ Ǒ ǒ Ǔ ǔ Ǖ ǖ\n",
	"Ǘ ǘ Ǚ ǚ Ǜ ǜ Ǟ ǟ Ǣ ǣ Ǥ ǥ Ǧ ǧ Ǩ ǩ Ǫ ǫ Ǭ ǭ Ǯ ǯ ǰ Ǵ ǵ Ǹ ǹ Ǻ ǻ Ǽ ǽ Ǿ ǿ Ȓ ȓ Ș ș Ț ț Ȟ ȟ ȧ Ȩ ȩ Ȫ ȫ Ȭ ȭ\n",
	"Ȯ ȯ Ȱ ȱ Ȳ ȳ ə ɨ ʒ Ḃ ḃ Ḇ ḇ Ḋ ḋ Ḍ ḍ Ḏ ḏ Ḑ ḑ ḗ Ḝ ḝ Ḟ ḟ Ḡ ḡ Ḣ ḣ Ḥ ḥ Ḧ ḧ Ḩ ḩ Ḫ ḫ ḯ Ḱ ḱ Ḳ ḳ Ḵ ḵ Ḷ ḷ Ḻ ḻ Ṁ\n",
	"ṁ Ṃ ṃ Ṅ ṅ Ṇ ṇ Ṉ ṉ Ṓ ṓ Ṕ ṕ Ṗ ṗ Ṙ ṙ Ṛ ṛ Ṟ ṟ Ṡ ṡ Ṣ ṣ Ṫ ṫ Ṭ ṭ Ṯ ṯ Ẁ ẁ Ẃ ẃ Ẅ ẅ Ẇ ẇ Ẍ ẍ Ẏ ẏ Ẑ ẑ Ẓ ẓ Ẕ ẕ\n",
	"ẖ ẗ ẞ Ạ ạ Ả ả Ấ ấ Ầ ầ Ẩ ẩ Ẫ ẫ Ậ ậ Ắ ắ Ằ ằ Ẳ ẳ Ẵ ẵ Ặ ặ Ẹ ẹ Ẻ ẻ Ẽ ẽ Ế ế Ề ề Ể ể Ễ ễ Ệ ệ Ỉ ỉ Ị ị Ọ ọ Ỏ ỏ Ố\n",
	"ố Ồ ồ Ổ ổ Ỗ ỗ Ộ ộ Ớ ớ Ờ ờ Ở ở Ỡ ỡ Ợ ợ Ụ ụ Ủ ủ Ứ ứ Ừ ừ Ử ử Ữ ữ Ự ự Ỳ ỳ Ỵ ỵ Ỷ ỷ Ỹ ỹ\n",
	"Sequences\n",
	"A̋ C̀ C̄ C̆ C̈ C̕ C̣ C̦ C̨̆ D̂ F̀ F̄ G̀ H̄ H̦ H̱ J́ J̌ K̀ K̂ K̄ K̇ K̕ K̛ K̦ K͟H\n",
	"K͟h L̂ L̥ L̥̄ L̦ M̀ M̂ M̆ M̐ N̂ N̄ N̆ N̦ P̀ P̄ P̕ P̣ R̆ R̥ R̥̄ S̀ S̄ S̛̄ S̱ T̀ T̄\n",
	"T̈ T̕ T̛ U̇ Z̀ Z̄ Z̆ Z̈ Z̧ a̋ c̀ c̄ c̆ c̈ c̕ c̣ c̦ c̨̆ d̂ f̀ f̄ g̀ h̄ h̦ j́ k̀\n",
	"k̂ k̄ k̇ k̕ k̛ k̦ k͟h l̂ l̥ l̥̄ l̦ m̀ m̂ m̆ m̐ n̂ n̄ n̆ n̦ p̀ p̄ p̕ p̣ r̆ r̥ r̥̄\n",
	"s̀ s̄ s̛̄ s̱ t̀ t̄ t̕ t̛ u̇ z̀ z̄ z̆ z̈ z̧ Ç̆ Û̄ ç̆ û̄ ÿ́ Č̕ Č̣ č̕ č̣ ē̍ Ī́ ī́\n",
	"ō̍ Ž̦ Ž̧ ž̦ ž̧ Ḳ̄ ḳ̄ Ṣ̄ ṣ̄ Ṭ̄ ṭ̄ Ạ̈ ạ̈ Ọ̈ ọ̈ Ụ̄ Ụ̈ ụ̄ ụ̈\n",
	"bnlreq; Non-Letters N1 (normative)\n",
	"  ' , - . ` ~ ¨ ´ · ʹ ʺ ʾ ʿ ˈ ˌ ’ ‡\n",
	"bnl; Non-Letters N2 (normative)\n",
	"! \" # $ % & ( ) * + / 0 1 2 3 4 5 6 7 8 9 : ; < = >\n",
	"? @ [ \\ ] ^ _ { | } ¡ ¢ £ ¥ § © ª « ¬ ® ¯ ° ± ² ³ µ\n",
	"¶ ¹ º » ¿ × ÷ €\n",
	"bnlopt; Non-Letters N3 (normative)\n",
	"¤ ¦ ¸ ¼ ½ ¾\n",
	"gl; Greek Letters (extended)\n",
	"Ά Έ Ή Ί Ό Ύ Ώ ΐ Α Β Γ Δ Ε Ζ Η Θ Ι Κ Λ Μ Ν Ξ Ο Π Ρ Σ\n",
	"Τ Υ Φ Χ Ψ Ω Ϊ Ϋ ά έ ή ί ΰ α β γ δ ε ζ η θ ι κ λ μ ν\n",
	"ξ ο π ρ ς σ τ υ φ χ ψ ω ϊ ϋ ό ύ ώ\n",
	"cl; Cyrillic Letters (extended)\n",
	"Ѝ А Б В Г Д Е Ж З И Й К Л М Н О П Р С Т У Ф Х Ц Ч Ш\n",
	"Щ Ъ Ь Ю Я а б в г д е ж з и й к л м н о п р с т у ф\n",
	"х ц ч ш щ ъ ь ю я ѝ\n",
	"enl; Non-Letters E1 (extended)\n",
	"ƒ ʰ ʳ ˆ ˜ ˢ ᵈ ᵗ ‘ ‚ “ ” „ † … ‰ ′ ″ ‹ › ⁰ ⁴ ⁵ ⁶ ⁷ ⁸\n",
	"⁹ ⁿ ₀ ₁ ₂ ₃ ₄ ₅ ₆ ₇ ₈ ₉ ™ ∞ ≤ ≥\n",
	"Additional non-letters (not included in DIN 91379): – — •�",
}, "")

// TestDin91379AgainstTheAwtReference is GlyphLayoutDin91379Test, compared with
// the PDF it renders against.
func TestDin91379AgainstTheAwtReference(t *testing.T) {
	compareWithReference(t, "awt-GlyphLayoutDIN91379.txt", din91379Page, map[int]string{
		17: "the sequence j-acute. The platform substitutes U+0237 LATIN SMALL " +
			"LETTER DOTLESS J for the j, so the accent does not land on the dot, " +
			"and then raises it differently. That is Arimo's \"ccmp\" feature, " +
			"which is two chained contextual lookups -- GSUB lookup type 6, which " +
			"PDFBox's reader drops. The port asks for ccmp and gets nothing back.",
	})
}

// din91379Page shows the character list one line at a time, which is what
// showComposites does with a string that has newlines in it.
func din91379Page(t *testing.T, document *pdmodel.PDDocument,
	stream *pdmodel.PDPageContentStream) {
	t.Helper()
	arimo := loadLayoutFont(t, document, "Arimo-Regular.ttf")
	processor := glyphlayout.NewProcessor()
	const size = 12
	const x = float32(12)
	y := float32(780)
	for _, line := range strings.Split(latinCharsDin91379, "\n") {
		if line == "" {
			continue
		}
		y = showComposites(t, stream, processor, arimo, size, x, y, line)
	}
}
