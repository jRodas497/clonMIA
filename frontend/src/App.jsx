import Editor from '@monaco-editor/react'
import FileInput from './components/FileInput'
import { useRef, useState} from 'react'

function App() {
  const editorRef = useRef(null)
  const consolaRef = useRef(null)

  const [ setEntradaFile ] = useState("")

  const handleEditor = (editor, id) => {
    if(id == "editor" ) {
        editorRef.current = editor
    }else if(id == "consola") {
        consolaRef.current = editor
    }
  }

  const confirmarRmdisk = (entrada) => entrada;

  const analizar = async () => {
    const confirmar = window.confirm('¿Está seguro de ejecutar los comandos?');
    if (!confirmar) {
      return;
    }
    var entrada = editorRef.current.getValue();
    const entradaFiltrada = confirmarRmdisk(entrada);
    const response = await fetch('http://localhost:3000/mia', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ comando: entradaFiltrada }),
    });
    const data = await response.json();
    consolaRef.current.setValue(data.resultados.join('\n'));
  }

  return (
    <div className="h-screen flex flex-col text-center justify-center">
      <section className='flex flex-col text-center justify-center'>
        <h1 className='m-4 text-2xl'>Entrada</h1>
        <div className='flex justify-center'>
          <Editor className='rounded-md'
              height="25vh" 
              width="55%"
              theme='vs-dark'
              defaultLanguage='cpp'
              defaultValue=''
              options={{
                scrollBeyondLastLine:false,
                fontSize:"16px"
              }}
              onMount={(editor) => handleEditor(editor, "editor")}
          />
        </div>
        <FileInput texto={setEntradaFile} editor={editorRef} />
        <button className='my-6 mx-auto p-2 rounded-md bg-btn w-1/12 text-xl font-bold text-white hover:bg-btn-osc'
          onClick={analizar}
        >
          Ejecutar
        </button>
      </section>
      <section className='flex flex-col text-center justify-center'>
        <h1 className='m-4 text-2xl'>Salida</h1>
        <div className='flex justify-center'>
          <Editor className='rounded-md'
              height="25vh" 
              width="55%"
              theme='vs-dark'
              defaultLanguage='cpp'
              defaultValue=''
              options={{
                scrollBeyondLastLine:false,
                fontSize:"16px",
                readOnly: true
              }}
              onMount={(editor) => handleEditor(editor, "consola")}
          />
        </div>
      </section>
    </div>
  )
}

export default App
