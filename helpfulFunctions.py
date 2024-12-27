
import json
def createUVsForBlockTextures():
    """
    Creates a map for all blocks and their corresponding faces to the current texture map.
    """
    textureWidth = 96.0
    textureHeight = 48.0
    res = []

    for i in range(0,int(textureHeight)):
        res.append([
            [(0*16)/textureWidth,(i*16)/textureHeight,((0+1)*16)/textureWidth,((i+1)*16)/textureHeight],
            [(1*16)/textureWidth,(i*16)/textureHeight,((1+1)*16)/textureWidth,((i+1)*16)/textureHeight],
            [(2*16)/textureWidth,(i*16)/textureHeight,((2+1)*16)/textureWidth,((i+1)*16)/textureHeight],
            [(3*16)/textureWidth,(i*16)/textureHeight,((3+1)*16)/textureWidth,((i+1)*16)/textureHeight],
            [(4*16)/textureWidth,(i*16)/textureHeight,((4+1)*16)/textureWidth,((i+1)*16)/textureHeight],
            [(5*16)/textureWidth,(i*16)/textureHeight,((5+1)*16)/textureWidth,((i+1)*16)/textureHeight]
        ])
        i += 16
    res = json.dumps(res)
    res = res.replace('[', '{').replace(']', '}')
    return res


def rgba(r,g,b,a=-1):
    """
    Turns an rgba value into a 0.0-1.0 normalized color value that OpenGL uses.
    """
    if a == -1 :
        #RGB value
        return [round(r/255,2),round(g/255,2),round(b/255,2)]
    return [round(r/255,2),round(g/255,2),round(b/255,2),float(a)]

#print(rgba(120, 167, 255,1))
print(createUVsForBlockTextures())