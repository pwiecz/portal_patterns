#version 150

uniform mat4 Matrix;

in vec2 Position;

out vec2 Frag_UV;

void main()
{
	vec4 pos = vec4(Position, 0.0, 1.0);
	Frag_UV = Position;
	gl_Position = Matrix * pos;
}