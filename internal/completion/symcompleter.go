package completion

import (
	"go/ast"
	"go/doc"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

func (c Completer) completeSymbol(pkg godoc.PackageInfo, partialSymbol string) (matched bool) {
	dlog.Printf("completing symbols matching %q", partialSymbol)

	// CONSTS & VARS
	pkgDoc := pkg.Doc()
	values := make([]*doc.Value, 0, len(pkgDoc.Consts)+len(pkgDoc.Vars))
	values = append(values, pkgDoc.Consts...)
	values = append(values, pkgDoc.Vars...)
	tag := TagConsts
	for i, value := range values {
		if i == len(pkgDoc.Consts) {
			tag = TagVars
		}
		tag := tag
		var passName bool
		for i, name := range value.Names {
			// The first name represents the group declaration, and
			// so we will suggest it with passName=false under the
			// consts or vars tag. This will cause it to appear
			// with an elipses (...) if there are additional names
			// in the group, just as go doc displays them.
			if i == 0 {
				// We only add untyped values to the consts and
				// vars tags. Typed values appear under the
				// types tag with their respective type.
				if !pkg.IsTypedValue(value) {
					matched = c.suggestIfMatchPrefix(pkg, partialSymbol, name, value.Doc, value.Decl, passName, WithTag(tag)) || matched
				}
				// We will also suggest this name and all
				// others under the all-consts or all-vars
				// tags. With passName=true there will be no
				// elipses and it will allow the subsequent
				// names to be properly rendered.
				tag = "all-" + tag
				passName = true
			}
			matched = c.suggestIfMatchPrefix(pkg, partialSymbol, name, value.Doc, value.Decl, passName, WithTag(tag)) || matched
		}
	}

	// FUNCS
	for _, fnc := range pkgDoc.Funcs {
		if pkg.IsConstructor(fnc) {
			// Constructors are shown under the types tag with
			// their respective type.
			continue
		}
		matched = c.suggestIfMatchPrefix(pkg, partialSymbol, fnc.Name, fnc.Doc, fnc.Decl, false, WithTag(TagFuncs)) || matched
	}

	// TYPES
	for _, typ := range pkgDoc.Types {
		// Suggest the type itself.
		typSpec := pkg.FindTypeSpec(typ.Decl, typ.Name)
		matched = c.suggestIfMatchPrefix(pkg, partialSymbol, typ.Name, typ.Doc, typSpec, false, WithTag(TagTypes)) || matched

		// Typed consts and vars.
		values := make([]*doc.Value, 0, len(typ.Consts)+len(typ.Vars))
		values = append(values, typ.Consts...)
		values = append(values, typ.Vars...)
		for _, value := range values {
			for _, name := range value.Names {
				matched = c.suggestIfMatchPrefix(pkg, partialSymbol, name, value.Doc, value.Decl, false, WithTag(TagTypes), WithDisplayIndent(true)) || matched
				// Remaining names were already suggested under
				// all-consts and all-vars above.
				break
			}
		}

		// Constructors
		for _, fnc := range typ.Funcs {
			matched = c.suggestIfMatchPrefix(pkg, partialSymbol, fnc.Name, fnc.Doc, fnc.Decl, false, WithTag(TagTypes), WithDisplayIndent(true)) || matched
		}

		if !c.IsExported(typ.Name) {
			// Don't suggest the raw methods of unexported types.
			continue
		}

		// Methods without the preceding `<type>.`
		for _, method := range typ.Methods {
			matched = c.suggestIfMatchPrefix(pkg, partialSymbol, method.Name, method.Doc, method.Decl, false, WithTag(TagMethods)) || matched
		}
	}

	return
}

func (c Completer) completeMethodOrField(pkg godoc.PackageInfo, symbol, partial string) (matched bool) {
	dlog.Printf("completing methods and fields for %q matching %q", symbol, partial)
	// We had <sym>.<method|field> so we must have a type.
	//
	// Search all types for matching symbols.
	//
	// Note that due to go doc's forgiving case rules, we may match
	// more than one symbol.
	for _, typ := range pkg.Doc().Types {
		if !c.MatchPartial(symbol, typ.Name) {
			// Not a match for symbol, moving on...
			continue
		}
		typSpec := pkg.FindTypeSpec(typ.Decl, typ.Name)
		matched = c.completeTypeDotMethodOrField(pkg, typ, typSpec, partial) || matched
	}
	return matched
}

func (c Completer) completeTypeDotMethodOrField(pkg godoc.PackageInfo, docTyp *doc.Type, typSpec *ast.TypeSpec, partial string) (matched bool) {
	// We had <sym>.<method|field> so we must have a type.
	//
	// Search all types for matching symbols.
	//
	// Note that due to go doc's forgiving case rules, we may match
	// more than one symbol.

	// WithType ensures all matches have the `<type>.` prefix added.
	c.opts = append(c.opts, WithType(docTyp.Name))

	// Type Methods (<type>.<method>)
	for _, method := range docTyp.Methods {
		matched = c.suggestIfMatchPrefix(pkg, partial, method.Name, method.Doc, method.Decl, false, WithTag(TagTypeMethods)) || matched
	}

	// Interface and struct types require special handling.
	switch typ := typSpec.Type.(type) {
	case *ast.InterfaceType:
		// Search interface methods for partial matches.
		for _, iMethod := range typ.Methods.List {
			// This is an interface, so there can be only one name.
			if len(iMethod.Names) == 0 {
				continue
			}
			name := iMethod.Names[0].Name
			matched = c.suggestIfMatchPrefix(pkg, partial, name, iMethod.Doc.Text(), iMethod, false, WithTag(TagInterfaceMethods)) || matched
		}
		// An interface has no fields or other methods so we are done
		// with this type.
		return

	case *ast.StructType:
		// Search struct fields for partial matches.
		for _, field := range typ.Fields.List {
			docs := field.Doc.Text()
			for _, name := range field.Names {
				matched = c.suggestIfMatchPrefix(pkg, partial, name.Name, docs, field, false, WithTag(TagStructFields)) || matched
			}
		}
	}
	return matched
}

func (c Completer) suggestIfMatchPrefix(pkg godoc.PackageInfo, partial, name, docs string, node ast.Node, useName bool, opts ...MatchOption) bool {
	if !c.MatchPartial(partial, name) {
		return false
	}

	var olnName string
	if useName {
		olnName = name
	}
	display := pkg.OneLineNode(node, olnName)

	docs = firstSentence(docs)
	docs = strings.TrimPrefix(docs, name+" ")

	c.suggest(NewMatch(name,
		WithOpts(c.opts...),
		WithOpts(opts...),
		WithDisplay(display),
		WithDescription(docs),
	))

	return true
}
